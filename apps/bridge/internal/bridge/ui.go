package bridge

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	headTagPattern         = regexp.MustCompile(`(?i)<head[^>]*>`)
	cspMetaPattern         = regexp.MustCompile(`(?i)<meta\s+http-equiv="Content-Security-Policy"\s+content="[^"]*"\s*/?>`)
	moduleScriptTagPattern = regexp.MustCompile(`(?i)<script\s+type="module"[^>]*></script>`)
)

func (b *Bridge) uiRoot() (string, bool) {
	root := strings.TrimSpace(b.config.DesktopWebviewRoot)
	if root == "" {
		return "", false
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return resolved, true
}

func normalizedUIPathPrefix(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "/ui"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func (b *Bridge) uiStatus() map[string]any {
	if root, ok := b.uiRoot(); ok {
		prefix := normalizedUIPathPrefix(b.config.UIPathPrefix)
		return map[string]any{
			"enabled":    true,
			"root":       root,
			"pathPrefix": prefix,
			"index":      prefix + "/index.html",
		}
	}
	return map[string]any{
		"enabled":    false,
		"root":       nil,
		"pathPrefix": nil,
		"index":      nil,
	}
}

func (b *Bridge) handleMobilePreload(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/codex-mobile/preload.js" {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return true
	}
	sendBody(w, r, http.StatusOK, []byte(buildMobilePreloadScript()), "application/javascript; charset=utf-8")
	return true
}

func (b *Bridge) handleUIAsset(w http.ResponseWriter, r *http.Request) bool {
	prefix := normalizedUIPathPrefix(b.config.UIPathPrefix)
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return true
	}
	root, ok := b.uiRoot()
	if !ok {
		sendText(w, http.StatusNotFound, "UI bundle not configured.\n")
		return true
	}

	relativePath := strings.TrimPrefix(r.URL.Path, prefix)
	relativePath = strings.TrimPrefix(relativePath, "/")
	if relativePath == "" {
		relativePath = "index.html"
	}

	requestedFile := filepath.Join(root, filepath.FromSlash(relativePath))
	resolvedFile, err := filepath.Abs(requestedFile)
	if err != nil {
		sendText(w, http.StatusForbidden, "Forbidden\n")
		return true
	}
	if !isPathInside(root, resolvedFile) && resolvedFile != filepath.Join(root, "index.html") {
		sendText(w, http.StatusForbidden, "Forbidden\n")
		return true
	}

	info, err := os.Stat(resolvedFile)
	if err != nil || info.IsDir() {
		sendText(w, http.StatusNotFound, "Not Found\n")
		return true
	}

	body, err := os.ReadFile(resolvedFile)
	if err != nil {
		sendText(w, http.StatusInternalServerError, "Failed to read UI asset.\n")
		return true
	}

	contentType := guessContentType(resolvedFile)
	if filepath.Base(resolvedFile) == "index.html" {
		body = []byte(rewriteDesktopIndexHTML(string(body), b.config, r))
		contentType = "text/html; charset=utf-8"
	} else if strings.EqualFold(filepath.Ext(resolvedFile), ".js") {
		body = []byte(rewriteUIJavaScriptAsset(relativePath, string(body)))
	}

	sendBody(w, r, http.StatusOK, body, contentType)
	return true
}

func isPathInside(parentPath string, childPath string) bool {
	relativePath, err := filepath.Rel(parentPath, childPath)
	if err != nil {
		return false
	}
	return relativePath != "." && relativePath != "" && !strings.HasPrefix(relativePath, "..") && !filepath.IsAbs(relativePath)
}

func buildUiCSP(cfg Config, request *http.Request) string {
	hostHeader := request.Host
	if hostHeader == "" {
		hostHeader = fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	}
	uiOrigin := "http://" + hostHeader
	if request.TLS != nil {
		uiOrigin = "https://" + hostHeader
	}
	return strings.Join([]string{
		"default-src 'self'",
		"img-src 'self' app: blob: data: https: http:",
		"child-src 'self' blob:",
		"frame-src 'self' blob:",
		"worker-src 'self' blob:",
		"script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"font-src 'self' data:",
		"media-src 'self' blob: data:",
		fmt.Sprintf("connect-src 'self' %s ws: wss: blob: data: sentry-ipc:", uiOrigin),
	}, "; ")
}

func rewriteDesktopIndexHTML(source string, cfg Config, request *http.Request) string {
	prefix := normalizedUIPathPrefix(cfg.UIPathPrefix)
	baseTag := fmt.Sprintf(`<base href="%s/">`, escapeHTMLAttribute(prefix))
	cspTag := fmt.Sprintf(`<meta http-equiv="Content-Security-Policy" content="%s">`, escapeHTMLAttribute(buildUiCSP(cfg, request)))
	theme := "dark"
	if request.URL.Query().Get("codexTheme") == "light" {
		theme = "light"
	}
	preloadBootstrap := strings.Join([]string{
		"<script>",
		fmt.Sprintf("window.__codexMobileInitialTheme=%q;", theme),
		`window.codexWindowType="electron";`,
		`try{if(document&&document.documentElement){document.documentElement.setAttribute("data-codex-window-type","electron");}}catch(error){}`,
		"</script>",
		`<script src="/codex-mobile/preload.js"></script>`,
	}, "")

	result := source
	if strings.Contains(result, "<!-- PROD_BASE_TAG_HERE -->") {
		result = strings.Replace(result, "<!-- PROD_BASE_TAG_HERE -->", baseTag, 1)
	} else if headTagPattern.MatchString(result) {
		result = headTagPattern.ReplaceAllString(result, "$0"+baseTag)
	}

	if cspMetaPattern.MatchString(result) {
		result = cspMetaPattern.ReplaceAllString(result, cspTag)
	} else if strings.Contains(result, "<!-- PROD_CSP_TAG_HERE -->") {
		result = strings.Replace(result, "<!-- PROD_CSP_TAG_HERE -->", cspTag, 1)
	} else if headTagPattern.MatchString(result) {
		result = headTagPattern.ReplaceAllString(result, "$0"+cspTag)
	}

	if moduleScriptTagPattern.MatchString(result) {
		result = moduleScriptTagPattern.ReplaceAllString(result, preloadBootstrap+"$0")
	} else {
		result = strings.Replace(result, "</head>", preloadBootstrap+"</head>", 1)
	}

	return result
}

func guessContentType(filePath string) string {
	extension := strings.ToLower(filepath.Ext(filePath))
	switch extension {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	}
	if guessed := mime.TypeByExtension(extension); guessed != "" {
		return guessed
	}
	return "application/octet-stream"
}

func sendBody(w http.ResponseWriter, r *http.Request, statusCode int, body []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func escapeHTMLAttribute(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(value)
}

func buildMobilePreloadScript() string {
	return strings.TrimSpace(`
(function () {
  var currentTheme = window.__codexMobileInitialTheme || "dark";
  var listenersByWorker = {};
  var themeListeners = [];
  var ENABLED_STATSIG_GATES = { "1609556872": true };
  var directRpcMethods = [
    "active-workspace-roots",
    "account/read",
    "account/rateLimits/read",
    "account-info",
    "app/list",
    "codex-home",
    "collaborationMode/list",
    "config/read",
    "developer-instructions",
    "extension-info",
    "fast-mode-rollout-metrics",
    "get-configuration",
    "get-copilot-api-proxy-info",
    "get-global-state",
    "gh-cli-status",
    "gh-pr-status",
    "git-origins",
    "has-custom-cli-executable",
    "hotkey-window-hotkey-state",
    "ide-context",
    "inbox-items",
    "is-copilot-api-available",
    "list-automations",
    "list-pending-automation-run-threads",
    "list-pinned-threads",
    "local-custom-agents",
    "local-environments",
    "locale-info",
    "mcpServerStatus/list",
    "mcp-codex-config",
    "model/list",
    "open-in-targets",
    "os-info",
    "paths-exist",
    "recommended-skills",
    "remote-workspace-directory-entries",
    "set-configuration",
    "set-global-state",
    "set-pinned-threads-order",
    "set-thread-pinned",
    "skills/list",
    "thread-terminal-snapshot",
    "thread/list",
    "workspace-root-options",
    "worktree-shell-environment-config"
  ];
  var directRpcInflight = Object.create(null);
  var directRpcCache = Object.create(null);
  var pendingBridgeRequests = Object.create(null);
  var nextBridgeRequestNumber = 0;

  function cloneJson(value) {
    try { return JSON.parse(JSON.stringify(value)); } catch (error) {}
    try { return String(value); } catch (error) {}
    return "[unserializable]";
  }

  function post(kind, payload) {
    var bridge = window.ReactNativeWebView;
    if (!bridge || typeof bridge.postMessage !== "function") {
      return;
    }
    bridge.postMessage(JSON.stringify({
      __codexMobile: true,
      kind: kind,
      payload: cloneJson(payload)
    }));
  }

  function trace(tag, payload) {
    post("console", {
      trace: "thread-open",
      tag: tag,
      now: typeof performance !== "undefined" && performance && typeof performance.now === "function"
        ? Math.round(performance.now())
        : Date.now(),
      payload: cloneJson(payload || null)
    });
  }

  window.__codexMobileTrace = trace;

  function allocateBridgeRequestId(prefix) {
    nextBridgeRequestNumber += 1;
    return String(prefix || "bridge") + "-" + String(nextBridgeRequestNumber);
  }

  function resolveBridgeRequest(requestId, result) {
    if (!requestId) {
      return;
    }
    var resolver = pendingBridgeRequests[requestId];
    if (typeof resolver !== "function") {
      return;
    }
    delete pendingBridgeRequests[requestId];
    resolver(cloneJson(result ?? null));
  }

  function requestBridgeResult(kind, payload, prefix) {
    var requestId = allocateBridgeRequestId(prefix);
    return new Promise(function (resolve) {
      pendingBridgeRequests[requestId] = resolve;
      post(kind, Object.assign({ requestId: requestId }, payload || {}));
    });
  }

  function arrayRemove(values, target) {
    if (!Array.isArray(values)) { return; }
    var index = values.indexOf(target);
    if (index >= 0) { values.splice(index, 1); }
  }

  function getWorkerListeners(workerId) {
    if (!listenersByWorker[workerId]) {
      listenersByWorker[workerId] = [];
    }
    return listenersByWorker[workerId];
  }

  function emitWorkerMessage(workerId, payload) {
    var listeners = getWorkerListeners(workerId).slice();
    for (var i = 0; i < listeners.length; i += 1) {
      try { listeners[i](payload); } catch (error) {}
    }
  }

  function emitThemeChanged(nextTheme) {
    var listeners = themeListeners.slice();
    for (var i = 0; i < listeners.length; i += 1) {
      try { listeners[i](nextTheme); } catch (error) {}
    }
  }

  function dispatchHostMessage(message) {
    try {
      window.dispatchEvent(new MessageEvent("message", { data: message }));
    } catch (error) {
      if (document && typeof document.createEvent === "function") {
        var event = document.createEvent("MessageEvent");
        event.initMessageEvent("message", false, false, message, "", "", window, null);
        window.dispatchEvent(event);
      }
    }
  }

  function ensureWindowType() {
    window.codexWindowType = "electron";
    try {
      if (document && document.documentElement) {
        document.documentElement.setAttribute("data-codex-window-type", "electron");
      }
    } catch (error) {}
  }

  function applyThemeAttributes(theme) {
    try {
      if (document && document.documentElement) {
        document.documentElement.setAttribute("data-codex-mobile-theme", theme);
        document.documentElement.setAttribute("data-theme", theme);
        document.documentElement.setAttribute("data-color-mode", theme);
        document.documentElement.style.colorScheme = theme;
        if (document.documentElement.classList) {
          document.documentElement.classList.toggle("electron-dark", theme === "dark");
          document.documentElement.classList.toggle("electron-light", theme === "light");
        }
      }
      if (document && document.body) {
        document.body.setAttribute("data-codex-mobile-theme", theme);
      }
    } catch (error) {}
  }

  function installThemeSurfaceOverrides() {
    var style;
    if (window.__codexMobileThemeSurfaceOverridesInstalled) {
      return;
    }
    window.__codexMobileThemeSurfaceOverridesInstalled = true;
    try {
      style = document.createElement("style");
      style.setAttribute("data-codex-mobile-theme-overrides", "true");
      style.textContent = [
        'html[data-codex-window-type="electron"].electron-light,html[data-codex-window-type="electron"].electron-light body{--codex-mobile-light-sidebar-surface:#f4f4f1;background-color:var(--codex-mobile-light-sidebar-surface)!important;}',
        'html[data-codex-window-type="electron"].electron-light{--color-token-bg-primary:var(--codex-mobile-light-sidebar-surface)!important;--color-background-panel:var(--codex-mobile-light-sidebar-surface)!important;--color-background-elevated-primary:var(--vscode-editor-background,#ffffff)!important;--color-background-elevated-secondary:var(--codex-mobile-light-sidebar-surface)!important;--color-background-editor-opaque:var(--vscode-editor-background,#ffffff)!important;--color-background-surface-under:var(--vscode-editor-background,#ffffff)!important;--hotkey-window-home-shell-background:var(--codex-mobile-light-sidebar-surface)!important;--color-simple-scrim:rgba(250,250,248,0.88)!important;}',
        'html[data-codex-window-type="electron"].electron-light .window-fx-sidebar-surface,html[data-codex-window-type="electron"].electron-light .app-header-tint{background-color:var(--codex-mobile-light-sidebar-surface)!important;background-image:none!important;}',
        'html[data-codex-window-type="electron"].electron-light .main-surface{background-color:var(--vscode-editor-background,#ffffff)!important;}',
        'html[data-codex-window-type="electron"].electron-light [cmdk-root],html[data-codex-window-type="electron"].electron-light [data-cmdk-root]{backdrop-filter:none!important;-webkit-backdrop-filter:none!important;}'
      ].join("\n");
      (document.head || document.documentElement || document.body).appendChild(style);
    } catch (error) {}
  }

  var pageScrollResetScheduled = false;

  function resetPageScrollPosition() {
    try {
      if (document && document.body) {
        document.body.scrollTop = 0;
      }
      if (document && document.documentElement) {
        document.documentElement.scrollTop = 0;
      }
      if (document && document.scrollingElement) {
        document.scrollingElement.scrollTop = 0;
      }
      if (window.scrollX !== 0 || window.scrollY !== 0) {
        window.scrollTo(0, 0);
      }
    } catch (error) {}
  }

  function schedulePageScrollReset() {
    if (pageScrollResetScheduled) {
      return;
    }
    pageScrollResetScheduled = true;
    requestAnimationFrame(function () {
      pageScrollResetScheduled = false;
      resetPageScrollPosition();
    });
  }

  function installPageScrollLock() {
    if (window.__codexMobilePageScrollLockInstalled) {
      return;
    }
    window.__codexMobilePageScrollLockInstalled = true;
    try {
      var style = document.createElement("style");
      style.setAttribute("data-codex-mobile-page-scroll-lock", "true");
      style.textContent = "html,body{overflow:hidden!important;overscroll-behavior:none!important;}";
      (document.head || document.documentElement || document.body).appendChild(style);
    } catch (error) {}
    schedulePageScrollReset();
    window.addEventListener("scroll", schedulePageScrollReset, { passive: true });
    window.addEventListener("resize", schedulePageScrollReset, { passive: true });
    window.addEventListener("orientationchange", schedulePageScrollReset, { passive: true });
    window.addEventListener("focus", schedulePageScrollReset, true);
    document.addEventListener("focusin", schedulePageScrollReset, true);
    if (window.visualViewport) {
      window.visualViewport.addEventListener("scroll", schedulePageScrollReset, { passive: true });
      window.visualViewport.addEventListener("resize", schedulePageScrollReset, { passive: true });
    }
  }

  function resolveCssLengthPx(rawValue, fallbackPx) {
    if (typeof rawValue !== "string") {
      return fallbackPx;
    }
    var value = rawValue.trim();
    if (!value) {
      return fallbackPx;
    }
    if (/^-?\d+(\.\d+)?px$/.test(value)) {
      return parseFloat(value);
    }
    try {
      var probe = document.createElement("div");
      probe.style.position = "fixed";
      probe.style.visibility = "hidden";
      probe.style.pointerEvents = "none";
      probe.style.width = value;
      probe.style.height = "0";
      (document.body || document.documentElement).appendChild(probe);
      var measured = probe.getBoundingClientRect().width;
      if (probe.parentNode) {
        probe.parentNode.removeChild(probe);
      }
      if (isFinite(measured) && measured > 0) {
        return measured;
      }
    } catch (error) {}
    return fallbackPx;
  }

  function installSidebarLongPressContextMenu() {
    if (window.__codexMobileSidebarLongPressInstalled) {
      return;
    }
    window.__codexMobileSidebarLongPressInstalled = true;
    if (!(("ontouchstart" in window) || (navigator && navigator.maxTouchPoints > 0))) {
      return;
    }

    var LONG_PRESS_MS = 450;
    var MOVE_TOLERANCE_PX = 12;
    var activeTouch = null;
    var suppressClickUntil = 0;
    var suppressNativeContextMenuUntil = 0;

    function clearActiveTouch() {
      if (activeTouch && activeTouch.timerId) {
        window.clearTimeout(activeTouch.timerId);
      }
      activeTouch = null;
    }

    function getSidebarLongPressRegion() {
      var root = document.documentElement;
      var styles = window.getComputedStyle(root);
      var safeAreaLeft = resolveCssLengthPx(styles.getPropertyValue("--safe-area-left"), 0);
      var sidebarWidth = resolveCssLengthPx(styles.getPropertyValue("--spacing-token-sidebar"), 300);
      if (!(sidebarWidth > 0)) {
        sidebarWidth = resolveCssLengthPx(styles.getPropertyValue("--token-sidebar"), 300);
      }
      var toolbarHeight = resolveCssLengthPx(styles.getPropertyValue("--height-toolbar"), 0);
      return {
        left: safeAreaLeft,
        right: safeAreaLeft + sidebarWidth,
        top: toolbarHeight
      };
    }

    function isPointInsideSidebar(clientX, clientY) {
      var region = getSidebarLongPressRegion();
      return clientX >= region.left && clientX <= region.right && clientY >= region.top;
    }

    function isEligibleLongPressTarget(target) {
      if (!target || typeof target.closest !== "function") {
        return false;
      }
      if (target.closest("input,textarea,select,option,[contenteditable=''],[contenteditable='true'],[role='textbox']")) {
        return false;
      }
      return true;
    }

    function findTouchByIdentifier(touchList, identifier) {
      if (!touchList) {
        return null;
      }
      for (var index = 0; index < touchList.length; index += 1) {
        if (touchList[index].identifier === identifier) {
          return touchList[index];
        }
      }
      return null;
    }

    function fireSidebarContextMenu() {
      if (!activeTouch) {
        return;
      }
      var pointX = activeTouch.clientX;
      var pointY = activeTouch.clientY;
      var target = document.elementFromPoint(pointX, pointY) || activeTouch.target || document.body;
      suppressClickUntil = Date.now() + 700;
      suppressNativeContextMenuUntil = Date.now() + 1200;
      try {
        target.dispatchEvent(new MouseEvent("contextmenu", {
          bubbles: true,
          cancelable: true,
          view: window,
          clientX: pointX,
          clientY: pointY,
          screenX: pointX,
          screenY: pointY,
          button: 2,
          buttons: 2,
          detail: 0
        }));
      } catch (error) {}
      clearActiveTouch();
    }

    document.addEventListener("touchstart", function (event) {
      if (!event.changedTouches || event.changedTouches.length !== 1 || (event.touches && event.touches.length !== 1)) {
        clearActiveTouch();
        return;
      }
      var touch = event.changedTouches[0];
      if (!touch || !isPointInsideSidebar(touch.clientX, touch.clientY)) {
        clearActiveTouch();
        return;
      }
      var target = touch.target || event.target;
      if (!isEligibleLongPressTarget(target)) {
        clearActiveTouch();
        return;
      }
      clearActiveTouch();
      activeTouch = {
        identifier: touch.identifier,
        clientX: touch.clientX,
        clientY: touch.clientY,
        target: target,
        timerId: window.setTimeout(fireSidebarContextMenu, LONG_PRESS_MS)
      };
    }, { capture: true, passive: true });

    document.addEventListener("touchmove", function (event) {
      if (!activeTouch) {
        return;
      }
      var touch = findTouchByIdentifier(event.changedTouches, activeTouch.identifier) || findTouchByIdentifier(event.touches, activeTouch.identifier);
      if (!touch) {
        return;
      }
      var deltaX = touch.clientX - activeTouch.clientX;
      var deltaY = touch.clientY - activeTouch.clientY;
      if (!isPointInsideSidebar(touch.clientX, touch.clientY) || Math.sqrt(deltaX * deltaX + deltaY * deltaY) > MOVE_TOLERANCE_PX) {
        clearActiveTouch();
      }
    }, { capture: true, passive: true });

    document.addEventListener("touchend", function (event) {
      if (Date.now() < suppressClickUntil) {
        var completedTouch = event.changedTouches && event.changedTouches.length > 0 ? event.changedTouches[0] : null;
        if (!completedTouch || isPointInsideSidebar(completedTouch.clientX, completedTouch.clientY)) {
          event.preventDefault();
          event.stopPropagation();
        }
      }
      if (!activeTouch) {
        return;
      }
      var touch = findTouchByIdentifier(event.changedTouches, activeTouch.identifier);
      if (!touch) {
        return;
      }
      clearActiveTouch();
    }, { capture: true, passive: false });

    document.addEventListener("touchcancel", function () {
      clearActiveTouch();
    }, { capture: true, passive: true });

    document.addEventListener("contextmenu", function (event) {
      if (Date.now() >= suppressNativeContextMenuUntil || !event.isTrusted) {
        return;
      }
      if (isPointInsideSidebar(event.clientX, event.clientY)) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);

    document.addEventListener("click", function (event) {
      if (Date.now() >= suppressClickUntil) {
        return;
      }
      if (isPointInsideSidebar(event.clientX, event.clientY)) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);

    window.addEventListener("blur", clearActiveTouch, true);
    document.addEventListener("scroll", clearActiveTouch, true);
  }

  function getDirectRpcCacheKey(method, params) {
    return method + "::" + JSON.stringify(params || {});
  }

  function getDirectRpcCacheTtlMs(method, params) {
    if (method === "account/read" && (!params || params.refreshToken === false)) {
      return 60000;
    }
    return 0;
  }

  function isDirectRpcMethod(method) {
    return typeof method === "string" && directRpcMethods.indexOf(method) >= 0;
  }

  function buildMcpSuccessResponse(hostId, requestId, result) {
    return {
      type: "mcp-response",
      hostId: hostId,
      id: requestId,
      result: result,
      message: { id: requestId, result: result },
      response: { id: requestId, result: result }
    };
  }

  function directRpc(method, params) {
    var cacheKey = getDirectRpcCacheKey(method, params);
    var cacheTtlMs = getDirectRpcCacheTtlMs(method, params);
    var cachedEntry = cacheTtlMs > 0 ? directRpcCache[cacheKey] : null;
    if (cachedEntry && Date.now() - cachedEntry.createdAt < cacheTtlMs) {
      trace("direct-rpc.cache-hit", { method: method });
      return Promise.resolve(cloneJson(cachedEntry.result));
    }
    if (directRpcInflight[cacheKey]) {
      trace("direct-rpc.join-inflight", { method: method });
      return directRpcInflight[cacheKey].then(function (result) {
        return cloneJson(result);
      });
    }
    var request = fetch("/codex-mobile/rpc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ method: method, params: params || {} })
    }).then(async function (response) {
      var payload = null;
      try {
        payload = await response.json();
      } catch (error) {
        payload = null;
      }
      if (!response.ok) {
        var httpMessage = payload && payload.error ? payload.error : "Direct RPC failed with HTTP " + response.status;
        throw new Error(httpMessage);
      }
      if (payload && payload.ok === true) {
        return payload.result;
      }
      throw new Error(payload && payload.error ? payload.error : "Direct RPC failed");
    }).then(function (result) {
      if (cacheTtlMs > 0) {
        directRpcCache[cacheKey] = { createdAt: Date.now(), result: cloneJson(result) };
      }
      return result;
    }).finally(function () {
      if (directRpcInflight[cacheKey] === request) {
        delete directRpcInflight[cacheKey];
      }
    });
    directRpcInflight[cacheKey] = request;
    return request;
  }

  function maybeHandleDirectMcpRequest(message) {
    var request = message && message.type === "mcp-request" ? message.request : null;
    var hostId = message && typeof message.hostId === "string" ? message.hostId : "local";
    var requestId = request && Object.prototype.hasOwnProperty.call(request, "id") ? request.id : null;
    var method = request && typeof request.method === "string" ? request.method : null;
    if (hostId !== "local" || request == null || requestId == null || !isDirectRpcMethod(method)) {
      return false;
    }
    trace("direct-rpc.start", { method: method, requestId: requestId });
    directRpc(method, request.params).then(function (result) {
      trace("direct-rpc.done", { method: method, requestId: requestId });
      dispatchHostMessage(buildMcpSuccessResponse(hostId, requestId, result));
    }).catch(function () {
      trace("direct-rpc.fallback", { method: method, requestId: requestId });
      post("bridge-send-message", message);
    });
    return true;
  }

  function createStatsigReadyStub(user) {
    var activeUser = user || { userID: "" };
    var listeners = { values_updated: [] };
    function emit(eventName) {
      var handlers = listeners[eventName] || [];
      for (var index = 0; index < handlers.length; index += 1) {
        try { handlers[index](); } catch (error) {}
      }
    }
    function normalizeUser(nextUser) {
      return nextUser && typeof nextUser === "object" ? nextUser : { userID: "" };
    }
    return {
      loadingStatus: "Ready",
      initializeAsync: function () {
        return Promise.resolve({ success: true, source: "Bootstrap", duration: 0 });
      },
      initializeSync: function () {
        return { success: true, source: "Bootstrap", duration: 0 };
      },
      getContext: function () {
        return { user: activeUser };
      },
      updateUserAsync: function (nextUser) {
        activeUser = normalizeUser(nextUser);
        emit("values_updated");
        return Promise.resolve({ success: true, source: "Bootstrap", duration: 0 });
      },
      updateUserSync: function (nextUser) {
        activeUser = normalizeUser(nextUser);
        emit("values_updated");
        return { success: true, source: "Bootstrap", duration: 0 };
      },
      checkGate: function (name) { return !!ENABLED_STATSIG_GATES[name]; },
      getFeatureGate: function (name) {
        return { name: name, value: !!ENABLED_STATSIG_GATES[name], ruleID: "mobile-shell", details: { reason: "mobile-shell" } };
      },
      getDynamicConfig: function (name) {
        return { name: name, value: {}, ruleID: "mobile-shell", details: { reason: "mobile-shell" }, get: function (_key, fallback) { return fallback; } };
      },
      getExperiment: function (name) {
        return { name: name, value: {}, ruleID: "mobile-shell", details: { reason: "mobile-shell" }, get: function (_key, fallback) { return fallback; } };
      },
      getLayer: function (name) {
        return { name: name, ruleID: "mobile-shell", details: { reason: "mobile-shell" }, get: function (_key, fallback) { return fallback; } };
      },
      getParameterStore: function (name) {
        return { name: name, get: function (_key, fallback) { return fallback; } };
      },
      on: function (eventName, handler) {
        if (!listeners[eventName]) { listeners[eventName] = []; }
        listeners[eventName].push(handler);
      },
      off: function (eventName, handler) {
        if (!listeners[eventName]) { return; }
        arrayRemove(listeners[eventName], handler);
      },
      $on: function (eventName, handler) { this.on(eventName, handler); },
      $emt: function () {},
      flush: function () { return Promise.resolve(); },
      shutdown: function () { return Promise.resolve(); },
      dataAdapter: {
        attach: function () {},
        getDataSync: function () { return null; },
        getDataAsync: function () { return Promise.resolve(null); },
        setData: function () {},
        setDataLegacy: function () {},
        prefetchData: function () { return Promise.resolve(); }
      }
    };
  }

  function buildStatsigBootstrapPayload() {
    var featureGates = {};
    Object.keys(ENABLED_STATSIG_GATES).forEach(function (name) {
      featureGates[name] = {
        name: name,
        value: !!ENABLED_STATSIG_GATES[name],
        rule_id: "mobile-shell",
        secondary_exposures: []
      };
    });
    return {
      feature_gates: featureGates,
      dynamic_configs: {},
      layer_configs: {},
      sdkParams: {},
      has_updates: false,
      time: Date.now()
    };
  }

  function installStatsigReadyStub() {
    var sdkKey = "client-sYWqzCYMRkUg4DqqiZcR5DGTNl2iD7zNJY0HoeDLzxR";
    var globalStatsig = window.__STATSIG__ || (window.__STATSIG__ = {});
    var instances = globalStatsig.instances || (globalStatsig.instances = {});
    if (instances[sdkKey]) {
      return;
    }
    var stub = createStatsigReadyStub({ userID: "" });
    instances[sdkKey] = stub;
    globalStatsig.firstInstance = globalStatsig.firstInstance || stub;
    post("statsig-stub-installed", { sdkKey: sdkKey });
  }

  function isStatsigInitUrl(url) {
    if (typeof url !== "string" || url.length === 0) {
      return false;
    }
    try {
      var parsed = new URL(url, window.location.href);
      return parsed.hostname === "ab.chatgpt.com" && parsed.pathname === "/v1/initialize";
    } catch (error) {
      return false;
    }
  }

  function isCodexProtocolUrl(url) {
    if (typeof url !== "string" || url.length === 0) {
      return false;
    }
    try {
      var parsed = new URL(url, window.location.href);
      return parsed.protocol === "vscode:" && parsed.hostname === "codex";
    } catch (error) {
      return false;
    }
  }

  function isSentryIpcUrl(url) {
    if (typeof url !== "string" || url.length === 0) {
      return false;
    }
    try {
      return new URL(url, window.location.href).protocol === "sentry-ipc:";
    } catch (error) {
      return false;
    }
  }

  function parseRequestUrl(url) {
    if (typeof url !== "string" || url.length === 0) {
      return null;
    }
    try {
      return new URL(url, window.location.href);
    } catch (error) {
      return null;
    }
  }

  function normalizeRequestPathname(pathname) {
    if (typeof pathname !== "string" || pathname.length === 0) {
      return "/";
    }
    return pathname.replace(/\/{2,}/g, "/");
  }

  function isExternalConnectorDirectoryUrl(url) {
    var parsed = parseRequestUrl(url);
    if (!parsed || parsed.origin === window.location.origin) {
      return false;
    }
    var hostname = String(parsed.hostname || "").toLowerCase();
    if (hostname !== "chatgpt.com" && hostname !== "openai.com" && !hostname.endsWith(".chatgpt.com") && !hostname.endsWith(".openai.com")) {
      return false;
    }
    var pathname = normalizeRequestPathname(parsed.pathname || "/");
    if (pathname.indexOf("/backend-api/") !== 0 && pathname.indexOf("/aip/") !== 0) {
      return false;
    }
    return pathname.indexOf("/connectors/") >= 0 || pathname.indexOf("/app_connectors/") >= 0;
  }

  function buildEmptyConnectorDirectoryPayload(url) {
    var parsed = parseRequestUrl(url);
    var pathname = normalizeRequestPathname(parsed && parsed.pathname ? parsed.pathname : "/");
    var payload = {
      data: [],
      items: [],
      results: [],
      connectors: [],
      categories: [],
      hasMore: false,
      has_more: false,
      nextCursor: null,
      next_cursor: null,
      total: 0
    };
    if (pathname.indexOf("/directory/") >= 0 || pathname.endsWith("/list")) {
      return payload;
    }
    if (pathname.indexOf("/logo") >= 0) {
      return {
        base64: "",
        contentType: "image/svg+xml"
      };
    }
    return payload;
  }

  function getFetchUrl(input) {
    if (typeof input === "string") {
      return input;
    }
    if (input && typeof input.url === "string") {
      return input.url;
    }
    return "";
  }

  async function readFetchBodyText(input, init) {
    if (init && typeof init.body === "string") {
      return init.body;
    }
    if (input && typeof input.clone === "function") {
      try {
        return await input.clone().text();
      } catch (error) {}
    }
    return "";
  }

  async function interceptCodexProtocolFetch(input, init) {
    var parsed = new URL(getFetchUrl(input), window.location.href);
    var method = parsed.pathname.replace(/^\/+/, "") || null;
    if (!method) {
      return new Response(JSON.stringify({ error: "Missing Codex RPC method." }), {
        status: 400,
        headers: { "Content-Type": "application/json" }
      });
    }
    var params = {};
    var rawBody = await readFetchBodyText(input, init);
    if (rawBody) {
      try {
        params = JSON.parse(rawBody);
      } catch (error) {}
    }
    var response = await fetch("/codex-mobile/rpc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ method: method, params: params || {} }),
      signal: init && init.signal ? init.signal : input && input.signal ? input.signal : void 0
    });
    var payload = null;
    try {
      payload = await response.json();
    } catch (error) {
      payload = null;
    }
    var body = payload && payload.ok === true ? payload.result : payload;
    return new Response(JSON.stringify(body ?? null), {
      status: response.status,
      headers: { "Content-Type": "application/json" }
    });
  }

  function interceptStatsigFetch() {
    if (typeof window.fetch !== "function" || typeof Response === "undefined") {
      return;
    }
    var originalFetch = window.fetch.bind(window);
    window.fetch = function (input, init) {
      var url = getFetchUrl(input);
      if (isStatsigInitUrl(url)) {
        return Promise.resolve(new Response(JSON.stringify(buildStatsigBootstrapPayload()), {
          status: 200,
          headers: { "Content-Type": "application/json" }
        }));
      }
      if (isCodexProtocolUrl(url)) {
        return interceptCodexProtocolFetch(input, init);
      }
      if (isSentryIpcUrl(url)) {
        return Promise.resolve(new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" }
        }));
      }
      if (isExternalConnectorDirectoryUrl(url)) {
        post("connector-directory-stubbed", { url: url });
        return Promise.resolve(new Response(JSON.stringify(buildEmptyConnectorDirectoryPayload(url)), {
          status: 200,
          headers: { "Content-Type": "application/json" }
        }));
      }
      return originalFetch(input, init);
    };
  }

  window.__codexMobileDisableAutoFocus = true;
  window.__codexMobileTheme = currentTheme;
  window.__codexMobileHost = {
    dispatchHostMessage: function (message) {
      dispatchHostMessage(message);
    },
    dispatchWorkerMessage: function (workerId, payload) {
      emitWorkerMessage(workerId, payload);
    },
    resolveBridgeRequest: function (requestId, result) {
      resolveBridgeRequest(requestId, result);
    },
    updateTheme: function (nextTheme) {
      currentTheme = nextTheme;
      window.__codexMobileTheme = nextTheme;
      ensureWindowType();
      applyThemeAttributes(nextTheme);
      emitThemeChanged(nextTheme);
    }
  };

  window.electronBridge = {
    windowType: "electron",
    sendMessageFromView: function (message) {
      var request = message && message.type === "mcp-request" ? message.request : null;
      if (request && typeof request.method === "string") {
        trace("sendMessageFromView", { method: request.method, requestId: request.id || null, hostId: message.hostId || null });
      }
      if (maybeHandleDirectMcpRequest(message)) {
        return Promise.resolve();
      }
      post("bridge-send-message", message);
      return Promise.resolve();
    },
    getPathForFile: function () {
      return null;
    },
    sendWorkerMessageFromView: function (workerId, payload) {
      post("bridge-send-worker-message", { workerId: workerId, payload: payload });
      return Promise.resolve();
    },
    subscribeToWorkerMessages: function (workerId, handler) {
      var listeners = getWorkerListeners(workerId);
      listeners.push(handler);
      return function () {
        var nextListeners = listenersByWorker[workerId];
        if (!nextListeners) { return; }
        arrayRemove(nextListeners, handler);
        if (nextListeners.length === 0) {
          delete listenersByWorker[workerId];
        }
      };
    },
    showContextMenu: function (payload) {
      trace("showContextMenu", {
        itemCount: Array.isArray(payload) ? payload.length : null,
        payload: payload
      });
      return requestBridgeResult("bridge-show-context-menu", { items: payload }, "context-menu");
    },
    showApplicationMenu: function (menuId, x, y) {
      trace("showApplicationMenu", { menuId: menuId, x: x, y: y });
      return requestBridgeResult("bridge-show-application-menu", { menuId: menuId, x: x, y: y }, "application-menu");
    },
    getFastModeRolloutMetrics: function () {
      return Promise.resolve({ enabled: true, estimatedSavedMs: 0, rolloutCountWithCompletedTurns: 0 });
    },
    getSystemThemeVariant: function () {
      return window.__codexMobileTheme;
    },
    subscribeToSystemThemeVariant: function (handler) {
      themeListeners.push(handler);
      return function () {
        arrayRemove(themeListeners, handler);
      };
    },
    triggerSentryTestError: function () {
      return Promise.reject(new Error("Sentry test is not supported in Codex Mobile."));
    },
    getSentryInitOptions: function () {
      return {
        codexAppSessionId: "codex-mobile",
        appVersion: "26.323.20928",
        buildFlavor: "prod",
        buildNumber: "1173",
        productName: "Codex"
      };
    },
    getAppSessionId: function () {
      return "codex-mobile";
    },
    getBuildFlavor: function () {
      return "prod";
    }
  };

  installStatsigReadyStub();
  interceptStatsigFetch();
  ensureWindowType();
  installThemeSurfaceOverrides();
  applyThemeAttributes(currentTheme);
  installPageScrollLock();
  installSidebarLongPressContextMenu();
  post("preload-ready", { theme: currentTheme, href: window.location.href });
})();
`) + "\n"
}

func rewriteUIJavaScriptAsset(relativePath string, source string) string {
	if matched, _ := regexp.MatchString(`^assets/index-[^/]+\.js$`, relativePath); matched {
		next := strings.ReplaceAll(
			source,
			"let{client:x,isLoading:S}=(0,x9.useClientAsyncInit)(NWe,y,b),C,w;if",
			"let{client:x,isLoading:S}=(0,x9.useClientAsyncInit)(NWe,y,b),C,w;S=!1;if",
		)
		next = strings.ReplaceAll(
			next,
			"He=()=>{L(n.id),(0,Z.startTransition)(()=>{r?.(),i?.()})}",
			"He=()=>{L(n.id),(0,Z.startTransition)(()=>{r?.()}),i&&window.setTimeout(()=>{i?.()},0)}",
		)
		next = strings.ReplaceAll(
			next,
			"n.autoFocus&&r.focus();",
			"n.autoFocus&&window.__codexMobileDisableAutoFocus!==!0&&r.focus();",
		)
		next = strings.ReplaceAll(
			next,
			"i||requestAnimationFrame(()=>{e.focus()})",
			"i||window.__codexMobileDisableAutoFocus!==!0&&requestAnimationFrame(()=>{e.focus()})",
		)
		next = strings.ReplaceAll(
			next,
			"requestAnimationFrame(()=>{en.focus()})",
			"window.__codexMobileDisableAutoFocus!==!0&&requestAnimationFrame(()=>{en.focus()})",
		)
		next = strings.ReplaceAll(
			next,
			"let e=()=>{en.focus()};return window.addEventListener(`focus`,e),()=>{window.removeEventListener(`focus`,e)}",
			"let e=()=>{window.__codexMobileDisableAutoFocus!==!0&&en.focus()};return window.addEventListener(`focus`,e),()=>{window.removeEventListener(`focus`,e)}",
		)
		return next
	}

	if strings.HasPrefix(relativePath, "assets/use-auth-") && strings.HasSuffix(relativePath, ".js") {
		return strings.ReplaceAll(
			source,
			"if(e==null){d(null),c(!1);return}d(null),c(!0)",
			"if(e==null){d(null),c(!1);return}l==null?c(!0):c(!1)",
		)
	}

	if strings.HasPrefix(relativePath, "assets/general-settings-") && strings.HasSuffix(relativePath, ".js") {
		return strings.ReplaceAll(
			source,
			"r=n?.fast_mode===!0&&Ot(t)",
			"r=Ot(t)",
		)
	}

	if relativePath == "assets/chunk-OIYGIGL5-CmK-41Zj.js" {
		next := source
		next = strings.ReplaceAll(
			next,
			"function ve(){a(R(),`useNavigate() may be used only in the context of a <Router> component.`);let e=r.useContext(M),{basename:t,navigator:n}=r.useContext(P),{matches:i}=r.useContext(I),{pathname:s}=z(),c=JSON.stringify(k(i)),l=r.useRef(!1);return _e(()=>{l.current=!0}),r.useCallback((r,i={})=>{if(o(l.current,ge),!l.current)return;if(typeof r==`number`){n.go(r);return}let a=A(r,JSON.parse(c),s,i.relative===`path`);e==null&&t!==`/`&&(a.pathname=a.pathname===`/`?t:j([t,a.pathname])),(i.replace?n.replace:n.push)(a,i.state,i)},[t,n,c,s,e])}",
			"function ve(){a(R(),`useNavigate() may be used only in the context of a <Router> component.`);let e=r.useContext(M),{basename:t,navigator:n}=r.useContext(P),{matches:i}=r.useContext(I),{pathname:s}=z(),c=JSON.stringify(k(i)),l=r.useRef(!1);return _e(()=>{l.current=!0}),r.useCallback((r,i={})=>{if(window.__codexMobileTrace)try{window.__codexMobileTrace(`router.navigate.standard`,{to:typeof r==`number`?r:String(r),replace:!!i.replace,relative:i.relative??null,pathname:s})}catch{}if(o(l.current,ge),!l.current)return;if(typeof r==`number`){n.go(r);return}let a=A(r,JSON.parse(c),s,i.relative===`path`);e==null&&t!==`/`&&(a.pathname=a.pathname===`/`?t:j([t,a.pathname])),(i.replace?n.replace:n.push)(a,i.state,i)},[t,n,c,s,e])}",
		)
		next = strings.ReplaceAll(
			next,
			"function Fe(){let{router:e}=ke(`useNavigate`),t=W(`useNavigate`),n=r.useRef(!1);return _e(()=>{n.current=!0}),r.useCallback(async(r,i={})=>{o(n.current,ge),n.current&&(typeof r==`number`?e.navigate(r):await e.navigate(r,{fromRouteId:t,...i}))},[e,t])}",
			"function Fe(){let{router:e}=ke(`useNavigate`),t=W(`useNavigate`),n=r.useRef(!1);return _e(()=>{n.current=!0}),r.useCallback(async(r,i={})=>{if(window.__codexMobileTrace)try{window.__codexMobileTrace(`router.navigate.data`,{to:typeof r==`number`?r:String(r),replace:!!i.replace,relative:i.relative??null,fromRouteId:t})}catch{}o(n.current,ge),n.current&&(typeof r==`number`?e.navigate(r):await e.navigate(r,{fromRouteId:t,...i}))},[e,t])}",
		)
		next = strings.ReplaceAll(
			next,
			"function ze({basename:e,children:t,initialEntries:n,initialIndex:a}){let o=r.useRef();o.current??=i({initialEntries:n,initialIndex:a,v5Compat:!0});let s=o.current,[c,l]=r.useState({action:s.action,location:s.location}),u=r.useCallback(e=>{r.startTransition(()=>l(e))},[l]);return r.useLayoutEffect(()=>s.listen(u),[s,u]),r.createElement(Ue,{basename:e,children:t,location:c.location,navigationType:c.action,navigator:s})}",
			"function ze({basename:e,children:t,initialEntries:n,initialIndex:a}){let o=r.useRef();o.current??=i({initialEntries:n,initialIndex:a,v5Compat:!0});let s=o.current,[c,l]=r.useState({action:s.action,location:s.location}),u=r.useCallback(e=>{if(window.__codexMobileTrace)try{window.__codexMobileTrace(`router.memory-listener`,{action:e?.action??null,pathname:e?.location?.pathname??null})}catch{}l(e)},[l]);return r.useLayoutEffect(()=>s.listen(u),[s,u]),r.createElement(Ue,{basename:e,children:t,location:c.location,navigationType:c.action,navigator:s})}",
		)
		next = strings.ReplaceAll(
			next,
			"function St({basename:e,children:t,history:n}){let[i,a]=r.useState({action:n.action,location:n.location}),o=r.useCallback(e=>{r.startTransition(()=>a(e))},[a]);return r.useLayoutEffect(()=>n.listen(o),[n,o]),r.createElement(Ue,{basename:e,children:t,location:i.location,navigationType:i.action,navigator:n})}",
			"function St({basename:e,children:t,history:n}){let[i,a]=r.useState({action:n.action,location:n.location}),o=r.useCallback(e=>{if(window.__codexMobileTrace)try{window.__codexMobileTrace(`router.history-listener`,{action:e?.action??null,pathname:e?.location?.pathname??null})}catch{}a(e)},[a]);return r.useLayoutEffect(()=>n.listen(o),[n,o]),r.createElement(Ue,{basename:e,children:t,location:i.location,navigationType:i.action,navigator:n})}",
		)
		return next
	}

	return source
}
