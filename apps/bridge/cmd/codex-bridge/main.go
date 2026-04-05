package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/boomyao/codex-bridge/internal/auth"
	"github.com/boomyao/codex-bridge/internal/bridge"
	"github.com/boomyao/codex-bridge/internal/config"
	"github.com/boomyao/codex-bridge/internal/exposure"
	exposurelibp2p "github.com/boomyao/codex-bridge/internal/exposure/libp2p"
	exposurelocal "github.com/boomyao/codex-bridge/internal/exposure/local"
	exposurenoop "github.com/boomyao/codex-bridge/internal/exposure/noop"
	exposuretailnet "github.com/boomyao/codex-bridge/internal/exposure/tailnet"
	exposuretunnel "github.com/boomyao/codex-bridge/internal/exposure/tunnel"
	bridgeruntime "github.com/boomyao/codex-bridge/internal/runtime"
	"github.com/boomyao/codex-bridge/internal/runtimestore"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/term"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

type commandName string

const (
	commandStart       commandName = "start"
	commandPrintConfig commandName = "print-config"
	commandInitConfig  commandName = "init-config"
	commandStop        commandName = "stop"
	commandQR          commandName = "qr"

	macOSBundledCodexPath = "/Applications/Codex.app/Contents/Resources/codex"
	tailscaleDownloadURL  = "https://tailscale.com/download"
)

var (
	bundledDesktopWebviewAppAsarPath = "/Applications/Codex.app/Contents/Resources/app.asar"
	execLookPath                     = exec.LookPath
	execCommandContext               = exec.CommandContext
)

type stringSliceFlag struct {
	values []string
}

func (f *stringSliceFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	f.values = append(f.values, value)
	return nil
}

type boolSwitch struct {
	set   bool
	value bool
}

func (b *boolSwitch) String() string {
	if !b.set {
		return ""
	}
	if b.value {
		return "true"
	}
	return "false"
}

func (b *boolSwitch) Set(value string) error {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "1", "t", "true", "y", "yes", "on":
		b.value = true
	case "0", "f", "false", "n", "no", "off":
		b.value = false
	default:
		return fmt.Errorf("invalid boolean value %q", value)
	}
	b.set = true
	return nil
}

func (b *boolSwitch) IsBoolFlag() bool {
	return true
}

type commandOptions struct {
	configPath                     string
	foreground                     boolSwitch
	backgroundChild                bool
	startupReportFile              string
	daemonLogFile                  string
	runtimeMode                    string
	runtimeRemoteURL               string
	runtimeListenHost              string
	runtimeListenPort              int
	appServerBin                   string
	appServerArgs                  stringSliceFlag
	runtimeCodexHome               string
	autoRestart                    boolSwitch
	noAutoRestart                  boolSwitch
	restartDelayMS                 int
	exposureMode                   string
	exposureAutoStart              boolSwitch
	exposureTunnelSSHBinary        string
	exposureTunnelSSHDestination   string
	exposureTunnelSSHPort          int
	exposureTunnelRemotePort       int
	exposureTunnelPublicHost       string
	exposureTunnelPublicPort       int
	exposureTunnelPublicScheme     string
	exposureTunnelSSHArgs          stringSliceFlag
	exposureTunnelAutoRestart      boolSwitch
	exposureTunnelRestartDelayMS   int
	exposureTailnetSocket          string
	exposureTailnetHostname        string
	exposureTailnetAddressStrategy string
	authMode                       string
	authRequireApproval            boolSwitch
	authDeviceStorePath            string
	authPairingCodeTTLMS           int
	gatewayHost                    string
	gatewayPort                    int
	desktopWebviewRoot             string
	uiPathPrefix                   string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	logger := log.New(stderr, "", 0)

	command, options, err := parseCLI(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	switch command {
	case commandPrintConfig:
		return runPrintConfig(options, stdout, stderr)
	case commandInitConfig:
		return runInitConfig(options, stdout, stderr)
	case commandStop:
		return runStop(options, stdout, stderr)
	case commandQR:
		return runQR(options, stdout, stderr)
	case commandStart:
		return runStart(args, options, stderr, logger)
	default:
		fmt.Fprintf(stderr, "unsupported command %q\n", command)
		return 2
	}
}

func parseCLI(args []string) (commandName, commandOptions, error) {
	command := commandStart
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case string(commandStart), string(commandPrintConfig), string(commandInitConfig), string(commandStop), string(commandQR):
			command = commandName(args[0])
			flagArgs = args[1:]
		default:
			return "", commandOptions{}, fmt.Errorf("unknown command %q", args[0])
		}
	}

	options, err := parseOptions(string(command), flagArgs)
	if err != nil {
		return "", commandOptions{}, err
	}
	return command, options, nil
}

func parseOptions(name string, args []string) (commandOptions, error) {
	var options commandOptions

	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	flagSet.StringVar(&options.configPath, "config", config.DefaultConfigPath(), "path to config.json")
	flagSet.Var(&options.foreground, "foreground", "keep codex-bridge attached to this terminal")
	flagSet.StringVar(&options.runtimeMode, "runtime-mode", "", "runtime mode: managed or remote")
	flagSet.StringVar(&options.runtimeRemoteURL, "runtime-remote-url", "", "remote runtime websocket URL")
	flagSet.StringVar(&options.runtimeListenHost, "runtime-listen-host", "", "managed runtime listen host")
	flagSet.IntVar(&options.runtimeListenPort, "runtime-listen-port", 0, "managed runtime listen port")
	flagSet.StringVar(&options.appServerBin, "app-server-bin", "", "managed app-server executable")
	flagSet.Var(&options.appServerArgs, "app-server-arg", "extra app-server argument")
	flagSet.StringVar(&options.runtimeCodexHome, "runtime-codex-home", "", "managed runtime CODEX_HOME directory")
	flagSet.Var(&options.autoRestart, "auto-restart", "enable managed runtime restart")
	flagSet.Var(&options.noAutoRestart, "no-auto-restart", "disable managed runtime restart")
	flagSet.IntVar(&options.restartDelayMS, "restart-delay-ms", 0, "restart delay in milliseconds")
	flagSet.StringVar(&options.exposureMode, "exposure-mode", "", "exposure mode: none, local, tunnel, tailnet, or libp2p")
	flagSet.Var(&options.exposureAutoStart, "exposure-auto-start", "auto start exposure provider")
	flagSet.StringVar(&options.exposureTunnelSSHBinary, "exposure-tunnel-ssh-binary", "", "tunnel provider ssh binary")
	flagSet.StringVar(&options.exposureTunnelSSHDestination, "exposure-tunnel-ssh-destination", "", "tunnel provider ssh destination, for example user@example.com")
	flagSet.IntVar(&options.exposureTunnelSSHPort, "exposure-tunnel-ssh-port", 0, "tunnel provider ssh port")
	flagSet.IntVar(&options.exposureTunnelRemotePort, "exposure-tunnel-remote-port", 0, "tunnel provider remote forwarded port")
	flagSet.StringVar(&options.exposureTunnelPublicHost, "exposure-tunnel-public-host", "", "tunnel provider public host")
	flagSet.IntVar(&options.exposureTunnelPublicPort, "exposure-tunnel-public-port", 0, "tunnel provider public port")
	flagSet.StringVar(&options.exposureTunnelPublicScheme, "exposure-tunnel-public-scheme", "", "tunnel provider public scheme: http or https")
	flagSet.Var(&options.exposureTunnelSSHArgs, "exposure-tunnel-ssh-arg", "extra tunnel provider ssh argument")
	flagSet.Var(&options.exposureTunnelAutoRestart, "exposure-tunnel-auto-restart", "auto restart tunnel provider")
	flagSet.IntVar(&options.exposureTunnelRestartDelayMS, "exposure-tunnel-restart-delay-ms", 0, "tunnel provider restart delay in milliseconds")
	flagSet.StringVar(&options.exposureTailnetSocket, "exposure-tailnet-socket", "", "tailnet provider LocalAPI socket path")
	flagSet.StringVar(&options.exposureTailnetHostname, "exposure-tailnet-hostname", "", "tailnet provider hostname override")
	flagSet.StringVar(&options.exposureTailnetAddressStrategy, "exposure-tailnet-address-strategy", "", "tailnet provider address strategy: auto, dns, ipv4, or ipv6")
	flagSet.StringVar(&options.authMode, "auth-mode", "", "auth mode: none or device-token")
	flagSet.Var(&options.authRequireApproval, "auth-require-approval", "require approval for authenticated devices")
	flagSet.StringVar(&options.authDeviceStorePath, "auth-device-store-path", "", "device-token auth store path")
	flagSet.IntVar(&options.authPairingCodeTTLMS, "auth-pairing-code-ttl-ms", 0, "device-token pairing code TTL in milliseconds")
	flagSet.StringVar(&options.gatewayHost, "gateway-host", "", "gateway host")
	flagSet.IntVar(&options.gatewayPort, "gateway-port", 0, "gateway port")
	flagSet.StringVar(&options.desktopWebviewRoot, "desktop-webview-root", "", "desktop webview root")
	flagSet.StringVar(&options.uiPathPrefix, "ui-path-prefix", "", "UI path prefix")
	flagSet.BoolVar(&options.backgroundChild, "background-child", false, "internal: detached child process")
	flagSet.StringVar(&options.startupReportFile, "startup-report-file", "", "internal: detached startup report file")
	flagSet.StringVar(&options.daemonLogFile, "daemon-log-file", "", "internal: detached process log file")

	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(), "Usage:\n")
		fmt.Fprintf(flagSet.Output(), "  codex-bridge [start] [options]\n")
		fmt.Fprintf(flagSet.Output(), "  codex-bridge print-config [options]\n")
		fmt.Fprintf(flagSet.Output(), "  codex-bridge init-config [options]\n")
		fmt.Fprintf(flagSet.Output(), "  codex-bridge stop [options]\n")
		fmt.Fprintf(flagSet.Output(), "  codex-bridge qr [options]\n")
		flagSet.VisitAll(func(fl *flag.Flag) {
			if isInternalFlag(fl.Name) {
				return
			}
			fmt.Fprintf(flagSet.Output(), "  -%s", fl.Name)
			name, usage := flag.UnquoteUsage(fl)
			if len(name) > 0 {
				fmt.Fprintf(flagSet.Output(), " %s", name)
			}
			fmt.Fprintln(flagSet.Output())
			fmt.Fprintf(flagSet.Output(), "    \t%s", usage)
			if fl.DefValue != "" && fl.DefValue != "false" && fl.DefValue != "0" {
				fmt.Fprintf(flagSet.Output(), " (default %q)", fl.DefValue)
			}
			fmt.Fprintln(flagSet.Output())
		})
	}

	if err := flagSet.Parse(args); err != nil {
		return commandOptions{}, err
	}
	if flagSet.NArg() > 0 {
		return commandOptions{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flagSet.Args(), " "))
	}
	if options.autoRestart.set && options.noAutoRestart.set {
		return commandOptions{}, errors.New("--auto-restart and --no-auto-restart cannot be used together")
	}
	return options, nil
}

func isInternalFlag(name string) bool {
	switch strings.TrimSpace(name) {
	case "background-child", "startup-report-file", "daemon-log-file":
		return true
	default:
		return false
	}
}

func runPrintConfig(options commandOptions, stdout, stderr *os.File) int {
	cfg, _, err := loadEffectiveConfig(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	body = append(body, '\n')
	_, _ = stdout.Write(body)
	return 0
}

func runInitConfig(options commandOptions, stdout, stderr *os.File) int {
	cfg, configPath, err := loadInitConfig(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "wrote %s\n", configPath)
	return 0
}

func runStop(options commandOptions, stdout, stderr *os.File) int {
	configPath := strings.TrimSpace(options.configPath)
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	pidPath := bridgePIDFilePath(absConfigPath)

	pid, err := readBridgePIDFile(pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "codex-bridge is not running for %s\n", absConfigPath)
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	running, err := processExists(pid)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !running {
		_ = os.Remove(pidPath)
		fmt.Fprintf(stdout, "removed stale pid file %s\n", pidPath)
		return 0
	}

	if err := signalBridgeStop(pid); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := waitForBridgeStop(pidPath, pid, 10*time.Second); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "stopped codex-bridge pid %d\n", pid)
	return 0
}

func runQR(options commandOptions, stdout, stderr *os.File) int {
	cfg, configPath, err := loadEffectiveConfig(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	configPath, err = filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	pidPath := bridgePIDFilePath(configPath)
	pid, err := readBridgePIDFile(pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "codex-bridge is not running for %s\n", configPath)
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	running, err := processExists(pid)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !running {
		_ = os.Remove(pidPath)
		fmt.Fprintf(stderr, "codex-bridge is not running for %s\n", configPath)
		return 1
	}

	baseURL, err := localBridgeHTTPURL(cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	payload, err := fetchRunningBridgeMobileEnrollment(baseURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	body, err := json.Marshal(payload.Payload)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	qr, err := buildStartupMobileQR(body)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if payload.Pairing != nil && strings.TrimSpace(payload.Pairing.Code) != "" {
		fmt.Fprintf(stdout, "mobile pairing code: %s\n", payload.Pairing.Code)
		fmt.Fprintf(stdout, "mobile pairing expires at: %s\n", payload.Pairing.ExpiresAt.Format(time.RFC3339))
	}
	fmt.Fprintf(stdout, "mobile enrollment payload: %s\n", string(body))
	fmt.Fprintf(stdout, "mobile QR:\n%s\n", qr)
	return 0
}

func runStart(args []string, options commandOptions, stderr *os.File, logger *log.Logger) int {
	cfg, configPath, err := loadEffectiveConfig(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	configPath, err = filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if !options.backgroundChild {
		cfg, err = prepareTailnetStartup(ctx, cfg, configPath, stderr, logger)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if shouldAutoBackground(options, stderr) {
			if exitCode, detached := runDetachedStart(ctx, args, cfg, configPath, stderr, logger); detached {
				return exitCode
			}
		}
	}

	pidGuard, err := acquireBridgePIDFile(bridgePIDFilePath(configPath), os.Getpid())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer pidGuard.Release()

	releaseRuntime, releaseRuntimeActivated, err := ensureReleaseRuntime(filepath.Dir(configPath), logger, cfg.Runtime.Mode == config.RuntimeModeManaged)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	runtimeConfig, err := buildRuntimeConfig(cfg, configPath, logger, releaseRuntime, releaseRuntimeActivated)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	runtimeManager := bridgeruntime.New(runtimeConfig)
	runtimeInfo, err := runtimeManager.Start(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	bridgeConfig, err := buildBridgeConfig(cfg, configPath, runtimeInfo.AppServerWebSocketURL, releaseRuntime, releaseRuntimeActivated)
	if err != nil {
		fmt.Fprintln(stderr, err)
		_ = runtimeManager.Stop(context.Background())
		return 1
	}

	bridgeServer := bridge.New(bridgeConfig, logger)
	bridgeServer.SetRuntimeStatus(bridge.RuntimeStatus{
		Mode:                  runtimeInfo.Mode,
		Ready:                 true,
		AppServerWebSocketURL: runtimeInfo.AppServerWebSocketURL,
	})

	authorizer, err := buildAuthorizer(cfg, configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		_ = runtimeManager.Stop(context.Background())
		return 1
	}
	bridgeServer.SetAuthorizer(authorizer)

	bridgeInfo, err := bridgeServer.Start(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		_ = runtimeManager.Stop(context.Background())
		return 1
	}

	exposureProvider, err := buildExposureProvider(cfg, logger)
	if err != nil {
		fmt.Fprintln(stderr, err)
		_ = bridgeServer.Close(context.Background())
		_ = runtimeManager.Stop(context.Background())
		return 1
	}

	target := exposure.Target{
		AppServerWebSocketURL: runtimeInfo.AppServerWebSocketURL,
		GatewayHTTPURL:        bridgeInfo.BridgeHTTPURL,
		GatewayWebSocketURL:   bridgeInfo.BridgeWebSocketURL,
	}

	var session *exposure.Session
	if cfg.Exposure.AutoStart {
		session, err = exposureProvider.Start(ctx, target)
		if err != nil {
			logger.Printf("[codex-bridge] exposure provider start failed: %v", err)
			if status, statusErr := exposureProvider.Status(ctx); statusErr == nil {
				session = status
			}
		}
	} else {
		session, err = exposureProvider.Status(ctx)
		if err != nil {
			logger.Printf("[codex-bridge] exposure provider status failed: %v", err)
		}
	}
	bridgeServer.SetExposureStatus(cfg.Exposure.Mode, session)
	startExposureStatusLoop(ctx, exposureProvider, bridgeServer, cfg.Exposure.Mode, logger)
	startupReport := buildStartupReport(bridgeServer, authorizer, bridgeInfo, options.daemonLogFile)
	logStartupReport(logger, startupReport)
	if err := writeStartupReport(options.startupReportFile, startupReport); err != nil {
		logger.Printf("%s [codex-bridge] failed to write startup report: %v", time.Now().UTC().Format(time.RFC3339), err)
	}

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if session != nil {
		_ = exposureProvider.Stop(stopCtx, session.ID)
	}
	if err := bridgeServer.Close(stopCtx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := runtimeManager.Stop(stopCtx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type startupReport struct {
	LogFile                 string `json:"logFile,omitempty"`
	BridgeHTTPURL           string `json:"bridgeHttpUrl,omitempty"`
	BridgeReadyURL          string `json:"bridgeReadyUrl,omitempty"`
	BridgeWebSocketURL      string `json:"bridgeWebSocketUrl,omitempty"`
	PairingCode             string `json:"pairingCode,omitempty"`
	PairingExpiresAt        string `json:"pairingExpiresAt,omitempty"`
	MobileEnrollmentPayload string `json:"mobileEnrollmentPayload,omitempty"`
	MobileQR                string `json:"mobileQR,omitempty"`
	MobileQRError           string `json:"mobileQRError,omitempty"`
}

type bridgePIDGuard struct {
	path string
	pid  int
}

func bridgePIDFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "codex-bridge.pid")
}

func acquireBridgePIDFile(path string, pid int) (bridgePIDGuard, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return bridgePIDGuard{}, errors.New("bridge pid file path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return bridgePIDGuard{}, fmt.Errorf("create bridge pid directory: %w", err)
	}

	existingPID, err := readBridgePIDFile(path)
	switch {
	case err == nil:
		running, runErr := processExists(existingPID)
		if runErr != nil {
			return bridgePIDGuard{}, fmt.Errorf("check existing bridge pid %d: %w", existingPID, runErr)
		}
		if running {
			return bridgePIDGuard{}, fmt.Errorf("codex-bridge is already running with pid %d", existingPID)
		}
		_ = os.Remove(path)
	case errors.Is(err, os.ErrNotExist):
	default:
		return bridgePIDGuard{}, err
	}

	if err := writeBridgePIDFile(path, pid); err != nil {
		return bridgePIDGuard{}, err
	}
	return bridgePIDGuard{path: path, pid: pid}, nil
}

func (g bridgePIDGuard) Release() {
	if strings.TrimSpace(g.path) == "" || g.pid <= 0 {
		return
	}
	currentPID, err := readBridgePIDFile(g.path)
	if err != nil || currentPID != g.pid {
		return
	}
	_ = os.Remove(g.path)
}

func readBridgePIDFile(path string) (int, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid bridge pid file %s", path)
	}
	return pid, nil
}

func writeBridgePIDFile(path string, pid int) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create bridge pid file: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("protect bridge pid file: %w", err)
	}
	if _, err := fmt.Fprintf(tempFile, "%d\n", pid); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write bridge pid file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close bridge pid file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("publish bridge pid file: %w", err)
	}
	return nil
}

func waitForBridgeStop(pidPath string, pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := processExists(pid)
		if err != nil {
			return err
		}
		if !running {
			_ = os.Remove(pidPath)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for codex-bridge pid %d to stop", pid)
}

func shouldAutoBackground(options commandOptions, stderr *os.File) bool {
	if options.backgroundChild || options.foreground.value {
		return false
	}
	if stderr == nil {
		return false
	}
	return term.IsTerminal(int(stderr.Fd()))
}

func runDetachedStart(ctx context.Context, args []string, cfg config.Config, configPath string, stderr *os.File, logger *log.Logger) (int, bool) {
	logFilePath, err := createDetachedLogFile(configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1, true
	}

	startupReportPath, err := createStartupReportPath()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1, true
	}

	cmd, err := buildDetachedCommand(args, logFilePath, startupReportPath)
	if err != nil {
		cleanupDetachedArtifacts(startupReportPath)
		fmt.Fprintln(stderr, err)
		return 1, true
	}

	if err := cmd.Start(); err != nil {
		cleanupDetachedArtifacts(startupReportPath)
		fmt.Fprintln(stderr, err)
		return 1, true
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	report, reportErr := waitForStartupReport(ctx, startupReportPath, 30*time.Second)
	cleanupDetachedArtifacts(startupReportPath)
	if reportErr != nil {
		if logger != nil {
			logger.Printf("%s [codex-bridge] launched in background with pid %d", time.Now().UTC().Format(time.RFC3339), pid)
			logger.Printf("%s [codex-bridge] startup is still in progress; logs: %s", time.Now().UTC().Format(time.RFC3339), logFilePath)
			logger.Printf("%s [codex-bridge] if startup succeeds, use `codex-bridge qr` to generate a fresh mobile QR", time.Now().UTC().Format(time.RFC3339))
		}
		return 0, true
	}

	if report.LogFile == "" {
		report.LogFile = logFilePath
	}
	if report.BridgeHTTPURL == "" {
		report.BridgeHTTPURL = fmt.Sprintf("http://%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	}
	logStartupReport(logger, report)
	if logger != nil {
		logger.Printf("%s [codex-bridge] codex-bridge is now running in the background with pid %d", time.Now().UTC().Format(time.RFC3339), pid)
		if report.LogFile != "" {
			logger.Printf("%s [codex-bridge] background log file: %s", time.Now().UTC().Format(time.RFC3339), report.LogFile)
		}
	}
	return 0, true
}

func startExposureStatusLoop(ctx context.Context, provider exposure.Provider, bridgeServer *bridge.Bridge, mode string, logger *log.Logger) {
	if provider == nil || bridgeServer == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			status, err := provider.Status(ctx)
			if err != nil {
				if logger != nil {
					logger.Printf("%s [codex-bridge] exposure provider status refresh failed: %v", time.Now().UTC().Format(time.RFC3339), err)
				}
			}
			bridgeServer.SetExposureStatus(mode, status)
		}
	}()
}

func prepareTailnetStartup(ctx context.Context, cfg config.Config, configPath string, stderr *os.File, logger *log.Logger) (config.Config, error) {
	if cfg.Exposure.Mode != config.ExposureModeTailnet {
		return cfg, nil
	}
	if err := waitForLocalTailnetReady(ctx, cfg, logger); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func waitForLocalTailnetReady(ctx context.Context, cfg config.Config, logger *log.Logger) error {
	client := &local.Client{}
	if socket := strings.TrimSpace(cfg.Exposure.Tailnet.Socket); socket != "" {
		client.Socket = socket
		client.UseSocketOnly = true
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	lastMessage := ""
	loginRequested := false
	appLaunchAttempted := false
	downloadOpened := false
	lastOpenedAuthURL := ""

	for {
		status, err := client.StatusWithoutPeers(ctx)
		installed := isTailscaleLikelyInstalled()
		if err == nil && status != nil && strings.TrimSpace(status.BackendState) == "Running" {
			if logger != nil {
				tailnetName := ""
				if status.CurrentTailnet != nil {
					tailnetName = strings.TrimSpace(status.CurrentTailnet.Name)
				}
				if tailnetName != "" {
					logger.Printf("%s [codex-bridge] local Tailscale is ready on tailnet %s", time.Now().UTC().Format(time.RFC3339), tailnetName)
				} else {
					logger.Printf("%s [codex-bridge] local Tailscale is ready", time.Now().UTC().Format(time.RFC3339))
				}
			}
			return nil
		}

		message := describeLocalTailnetWait(err, status, installed)
		if logger != nil && message != lastMessage {
			logger.Printf("%s [codex-bridge] %s", time.Now().UTC().Format(time.RFC3339), message)
			lastMessage = message
		}

		if err == nil && status != nil && strings.TrimSpace(status.BackendState) == "NeedsLogin" && !loginRequested {
			if startErr := client.StartLoginInteractive(ctx); startErr == nil {
				loginRequested = true
				if logger != nil {
					logger.Printf("%s [codex-bridge] requested interactive Tailscale login", time.Now().UTC().Format(time.RFC3339))
				}
			}
		}

		if err != nil {
			if !installed && !downloadOpened {
				downloadOpened = true
				if opened, openErr := openURLInBrowser(tailscaleDownloadURL); opened && openErr == nil && logger != nil {
					logger.Printf("%s [codex-bridge] opened Tailscale download page: %s", time.Now().UTC().Format(time.RFC3339), tailscaleDownloadURL)
				}
			}
			if installed && !appLaunchAttempted {
				appLaunchAttempted = true
				if launched, launchErr := tryLaunchLocalTailscaleApp(logger); launchErr == nil && launched && logger != nil {
					logger.Printf("%s [codex-bridge] launched Tailscale app, waiting for the local daemon to come online", time.Now().UTC().Format(time.RFC3339))
				}
			}
		}

		if err == nil && status != nil {
			authURL := strings.TrimSpace(status.AuthURL)
			if authURL != "" && authURL != lastOpenedAuthURL {
				if opened, openErr := openURLInBrowser(authURL); opened && openErr == nil {
					lastOpenedAuthURL = authURL
					if logger != nil {
						logger.Printf("%s [codex-bridge] opened Tailscale login URL: %s", time.Now().UTC().Format(time.RFC3339), authURL)
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func describeLocalTailnetWait(err error, status *ipnstate.Status, installed bool) string {
	if err != nil {
		if !installed {
			return fmt.Sprintf("Tailscale does not appear to be installed; install it from %s and sign in, then Codex Bridge will continue automatically (Ctrl+C to abort)", tailscaleDownloadURL)
		}
		return "waiting for local Tailscale daemon; start Tailscale and finish login if prompted (Ctrl+C to abort)"
	}
	if status == nil {
		return "waiting for local Tailscale daemon status (Ctrl+C to abort)"
	}
	backendState := strings.TrimSpace(status.BackendState)
	authURL := strings.TrimSpace(status.AuthURL)
	switch backendState {
	case "NeedsLogin":
		if authURL != "" {
			return fmt.Sprintf("waiting for local Tailscale login to complete; finish the browser flow at %s (Ctrl+C to abort)", authURL)
		}
		return "waiting for local Tailscale login to complete (Ctrl+C to abort)"
	case "NeedsMachineAuth":
		return "waiting for this machine to be approved in the tailnet admin console (Ctrl+C to abort)"
	case "Starting", "Stopped", "NoState", "":
		return fmt.Sprintf("waiting for local Tailscale to become ready (current state: %s, Ctrl+C to abort)", emptyIfBlank(backendState, "unknown"))
	default:
		return fmt.Sprintf("waiting for local Tailscale to become ready (current state: %s, Ctrl+C to abort)", backendState)
	}
}

func tryLaunchLocalTailscaleApp(logger *log.Logger) (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("open", "-g", "-a", "Tailscale")
		if err := cmd.Run(); err != nil {
			if logger != nil {
				logger.Printf("%s [codex-bridge] could not auto-launch Tailscale.app: %v", time.Now().UTC().Format(time.RFC3339), err)
			}
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}

func emptyIfBlank(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func isTailscaleLikelyInstalled() bool {
	switch runtime.GOOS {
	case "darwin":
		if _, err := os.Stat("/Applications/Tailscale.app"); err == nil {
			return true
		}
	}
	if _, err := exec.LookPath("tailscale"); err == nil {
		return true
	}
	return false
}

type localMobileEnrollmentResponse struct {
	OK      bool              `json:"ok"`
	Error   string            `json:"error,omitempty"`
	Pairing *auth.PairingInfo `json:"pairing,omitempty"`
	Payload map[string]any    `json:"payload,omitempty"`
}

func localBridgeHTTPURL(cfg config.Config) (string, error) {
	port := cfg.Gateway.Port
	if port <= 0 {
		return "", errors.New("gateway.port must be set to a fixed local port for codex-bridge qr")
	}
	host := strings.TrimSpace(cfg.Gateway.Host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

func fetchRunningBridgeMobileEnrollment(baseURL string) (localMobileEnrollmentResponse, error) {
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+bridge.LocalMobileEnrollmentPath,
		nil,
	)
	if err != nil {
		return localMobileEnrollmentResponse{}, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return localMobileEnrollmentResponse{}, fmt.Errorf("request running bridge mobile enrollment: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return localMobileEnrollmentResponse{}, fmt.Errorf("read running bridge mobile enrollment response: %w", err)
	}
	var payload localMobileEnrollmentResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		message := strings.TrimSpace(string(body))
		if response.StatusCode >= http.StatusBadRequest && message != "" {
			return localMobileEnrollmentResponse{}, errors.New("running bridge does not support `codex-bridge qr` yet; restart it with the updated binary")
		}
		return localMobileEnrollmentResponse{}, fmt.Errorf("decode running bridge mobile enrollment response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.OK {
		message := strings.TrimSpace(payload.Error)
		if message == "" {
			message = response.Status
		}
		return localMobileEnrollmentResponse{}, fmt.Errorf("running bridge mobile enrollment failed: %s", message)
	}
	if len(payload.Payload) == 0 {
		return localMobileEnrollmentResponse{}, errors.New("running bridge mobile enrollment response did not include a payload")
	}
	return payload, nil
}

func openURLInBrowser(rawURL string) (bool, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false, nil
	}
	switch runtime.GOOS {
	case "darwin":
		return true, exec.Command("open", rawURL).Run()
	case "linux":
		return true, exec.Command("xdg-open", rawURL).Run()
	case "windows":
		return true, exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Run()
	default:
		return false, nil
	}
}

func buildStartupReport(bridgeServer *bridge.Bridge, authorizer auth.Authorizer, bridgeInfo *bridge.Info, logFile string) startupReport {
	report := startupReport{
		LogFile: logFile,
	}
	if bridgeInfo != nil {
		report.BridgeHTTPURL = bridgeInfo.BridgeHTTPURL
		report.BridgeReadyURL = bridgeInfo.BridgeReadyURL
		report.BridgeWebSocketURL = bridgeInfo.BridgeWebSocketURL
	}
	if bridgeServer == nil {
		report.MobileQRError = "bridge server is not available"
		return report
	}
	deviceTokenAuthorizer, ok := authorizer.(*auth.DeviceTokenAuthorizer)
	if !ok {
		return report
	}

	pairing, err := deviceTokenAuthorizer.GeneratePairingCode()
	if err != nil {
		report.MobileQRError = fmt.Sprintf("failed to generate startup pairing code: %v", err)
		return report
	}
	report.PairingCode = pairing.Code
	report.PairingExpiresAt = pairing.ExpiresAt.Format(time.RFC3339)

	payload, err := bridgeServer.BuildMobileEnrollmentPayload(pairing.Code)
	if err != nil {
		report.MobileQRError = fmt.Sprintf("failed to build startup mobile enrollment payload: %v", err)
		return report
	}
	body, err := json.Marshal(payload)
	if err != nil {
		report.MobileQRError = fmt.Sprintf("failed to encode startup mobile enrollment payload: %v", err)
		return report
	}
	report.MobileEnrollmentPayload = string(body)

	qr, err := buildStartupMobileQR(body)
	if err != nil {
		report.MobileQRError = fmt.Sprintf("failed to build startup mobile QR: %v", err)
		return report
	}
	report.MobileQR = qr
	return report
}

func buildStartupMobileQR(body []byte) (string, error) {
	qr, err := qrcode.New(string(body), qrcode.Low)
	if err != nil {
		return "", err
	}
	return qr.ToSmallString(false), nil
}

func logStartupReport(logger *log.Logger, report startupReport) {
	if logger == nil {
		return
	}
	if report.PairingCode != "" {
		logger.Printf("%s [codex-bridge] mobile pairing code: %s", time.Now().UTC().Format(time.RFC3339), report.PairingCode)
	}
	if report.PairingExpiresAt != "" {
		logger.Printf("%s [codex-bridge] mobile pairing expires at: %s", time.Now().UTC().Format(time.RFC3339), report.PairingExpiresAt)
	}
	if report.MobileEnrollmentPayload != "" {
		logger.Printf("%s [codex-bridge] mobile enrollment payload: %s", time.Now().UTC().Format(time.RFC3339), report.MobileEnrollmentPayload)
	}
	if report.MobileQR != "" {
		logger.Printf("%s [codex-bridge] mobile QR:\n%s", time.Now().UTC().Format(time.RFC3339), report.MobileQR)
	}
	if report.MobileQRError != "" {
		logger.Printf("%s [codex-bridge] %s", time.Now().UTC().Format(time.RFC3339), report.MobileQRError)
	}
}

func createDetachedLogFile(configPath string) (string, error) {
	logsDir := filepath.Join(filepath.Dir(configPath), "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return "", fmt.Errorf("create log directory: %w", err)
	}
	fileName := fmt.Sprintf("codex-bridge-%s.log", time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(logsDir, fileName), nil
}

func createStartupReportPath() (string, error) {
	file, err := os.CreateTemp("", "codex-bridge-startup-report-*.json")
	if err != nil {
		return "", fmt.Errorf("create startup report file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close startup report file: %w", err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("prepare startup report file: %w", err)
	}
	return path, nil
}

func buildDetachedCommand(args []string, logFilePath string, startupReportPath string) (*exec.Cmd, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	if resolvedPath, resolveErr := filepath.EvalSymlinks(executablePath); resolveErr == nil {
		executablePath = resolvedPath
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open background log file: %w", err)
	}

	childArgs := append([]string{}, args...)
	childArgs = append(childArgs, "--background-child", "--startup-report-file", startupReportPath, "--daemon-log-file", logFilePath)

	cmd := exec.Command(executablePath, childArgs...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	configureDetachedProcess(cmd)
	return cmd, nil
}

func waitForStartupReport(ctx context.Context, path string, timeout time.Duration) (startupReport, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		report, err := readStartupReport(path)
		if err == nil {
			return report, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
				return startupReport{}, err
			}
		}
		select {
		case <-ctx.Done():
			return startupReport{}, ctx.Err()
		case <-deadline.C:
			return startupReport{}, context.DeadlineExceeded
		case <-ticker.C:
		}
	}
}

func readStartupReport(path string) (startupReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return startupReport{}, err
	}
	var report startupReport
	if err := json.Unmarshal(body, &report); err != nil {
		return startupReport{}, err
	}
	return report, nil
}

func writeStartupReport(path string, report startupReport) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal startup report: %w", err)
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create startup report temp file: %w", err)
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(body); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write startup report temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close startup report temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("publish startup report: %w", err)
	}
	return nil
}

func cleanupDetachedArtifacts(paths ...string) {
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		_ = os.Remove(trimmed)
	}
}

func loadEffectiveConfig(options commandOptions) (config.Config, string, error) {
	configPath := strings.TrimSpace(options.configPath)
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	fileConfig, err := config.LoadConfig(configPath)
	if err != nil {
		return config.Config{}, "", err
	}

	cfg, err := config.MergeConfigs(
		config.ConfigToMap(fileConfig),
		config.EnvConfig(),
		options.overrideMap(),
	)
	if err != nil {
		return config.Config{}, "", err
	}
	return cfg, configPath, nil
}

func loadInitConfig(options commandOptions) (config.Config, string, error) {
	configPath := strings.TrimSpace(options.configPath)
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	cfg, err := config.MergeConfigs(
		config.ConfigToMap(config.DefaultConfig()),
		config.EnvConfig(),
		options.overrideMap(),
	)
	if err != nil {
		return config.Config{}, "", err
	}
	return cfg, configPath, nil
}

func (o commandOptions) overrideMap() map[string]any {
	result := map[string]any{}
	runtimeCfg := map[string]any{}
	exposureCfg := map[string]any{}
	authCfg := map[string]any{}
	gatewayCfg := map[string]any{}

	if strings.TrimSpace(o.runtimeMode) != "" {
		runtimeCfg["mode"] = strings.TrimSpace(o.runtimeMode)
	}
	if strings.TrimSpace(o.runtimeRemoteURL) != "" {
		runtimeCfg["remoteUrl"] = strings.TrimSpace(o.runtimeRemoteURL)
	}
	if strings.TrimSpace(o.runtimeListenHost) != "" {
		runtimeCfg["listenHost"] = strings.TrimSpace(o.runtimeListenHost)
	}
	if o.runtimeListenPort > 0 {
		runtimeCfg["listenPort"] = o.runtimeListenPort
	}
	if strings.TrimSpace(o.appServerBin) != "" {
		runtimeCfg["appServerBin"] = strings.TrimSpace(o.appServerBin)
	}
	if len(o.appServerArgs.values) > 0 {
		runtimeCfg["appServerArgs"] = append([]string{}, o.appServerArgs.values...)
	}
	if strings.TrimSpace(o.runtimeCodexHome) != "" {
		runtimeCfg["codexHome"] = strings.TrimSpace(o.runtimeCodexHome)
	}
	if o.autoRestart.set {
		runtimeCfg["autoRestart"] = o.autoRestart.value
	}
	if o.noAutoRestart.set && o.noAutoRestart.value {
		runtimeCfg["autoRestart"] = false
	}
	if o.restartDelayMS > 0 {
		runtimeCfg["restartDelayMs"] = o.restartDelayMS
	}

	if strings.TrimSpace(o.exposureMode) != "" {
		exposureCfg["mode"] = strings.TrimSpace(o.exposureMode)
	}
	if o.exposureAutoStart.set {
		exposureCfg["autoStart"] = o.exposureAutoStart.value
	}
	tunnelCfg := map[string]any{}
	if strings.TrimSpace(o.exposureTunnelSSHBinary) != "" {
		tunnelCfg["sshBinary"] = strings.TrimSpace(o.exposureTunnelSSHBinary)
	}
	if strings.TrimSpace(o.exposureTunnelSSHDestination) != "" {
		tunnelCfg["sshDestination"] = strings.TrimSpace(o.exposureTunnelSSHDestination)
	}
	if o.exposureTunnelSSHPort > 0 {
		tunnelCfg["sshPort"] = o.exposureTunnelSSHPort
	}
	if o.exposureTunnelRemotePort > 0 {
		tunnelCfg["remotePort"] = o.exposureTunnelRemotePort
	}
	if strings.TrimSpace(o.exposureTunnelPublicHost) != "" {
		tunnelCfg["publicHost"] = strings.TrimSpace(o.exposureTunnelPublicHost)
	}
	if o.exposureTunnelPublicPort > 0 {
		tunnelCfg["publicPort"] = o.exposureTunnelPublicPort
	}
	if strings.TrimSpace(o.exposureTunnelPublicScheme) != "" {
		tunnelCfg["publicScheme"] = strings.TrimSpace(o.exposureTunnelPublicScheme)
	}
	if len(o.exposureTunnelSSHArgs.values) > 0 {
		tunnelCfg["sshArgs"] = append([]string{}, o.exposureTunnelSSHArgs.values...)
	}
	if o.exposureTunnelAutoRestart.set {
		tunnelCfg["autoRestart"] = o.exposureTunnelAutoRestart.value
	}
	if o.exposureTunnelRestartDelayMS > 0 {
		tunnelCfg["restartDelayMs"] = o.exposureTunnelRestartDelayMS
	}
	if len(tunnelCfg) > 0 {
		exposureCfg["tunnel"] = tunnelCfg
	}
	tailnetCfg := map[string]any{}
	if strings.TrimSpace(o.exposureTailnetSocket) != "" {
		tailnetCfg["socket"] = strings.TrimSpace(o.exposureTailnetSocket)
	}
	if strings.TrimSpace(o.exposureTailnetHostname) != "" {
		tailnetCfg["hostname"] = strings.TrimSpace(o.exposureTailnetHostname)
	}
	if strings.TrimSpace(o.exposureTailnetAddressStrategy) != "" {
		tailnetCfg["addressStrategy"] = strings.TrimSpace(o.exposureTailnetAddressStrategy)
	}
	if len(tailnetCfg) > 0 {
		exposureCfg["tailnet"] = tailnetCfg
	}

	if strings.TrimSpace(o.authMode) != "" {
		authCfg["mode"] = strings.TrimSpace(o.authMode)
	}
	if o.authRequireApproval.set {
		authCfg["requireApproval"] = o.authRequireApproval.value
	}
	if strings.TrimSpace(o.authDeviceStorePath) != "" {
		authCfg["deviceStorePath"] = strings.TrimSpace(o.authDeviceStorePath)
	}
	if o.authPairingCodeTTLMS > 0 {
		authCfg["pairingCodeTtlMs"] = o.authPairingCodeTTLMS
	}

	if strings.TrimSpace(o.gatewayHost) != "" {
		gatewayCfg["host"] = strings.TrimSpace(o.gatewayHost)
	}
	if o.gatewayPort > 0 {
		gatewayCfg["port"] = o.gatewayPort
	}
	if strings.TrimSpace(o.desktopWebviewRoot) != "" {
		gatewayCfg["desktopWebviewRoot"] = strings.TrimSpace(o.desktopWebviewRoot)
	}
	if strings.TrimSpace(o.uiPathPrefix) != "" {
		gatewayCfg["uiPathPrefix"] = strings.TrimSpace(o.uiPathPrefix)
	}

	if len(runtimeCfg) > 0 {
		result["runtime"] = runtimeCfg
	}
	if len(exposureCfg) > 0 {
		result["exposure"] = exposureCfg
	}
	if len(authCfg) > 0 {
		result["auth"] = authCfg
	}
	if len(gatewayCfg) > 0 {
		result["gateway"] = gatewayCfg
	}

	return result
}

func buildRuntimeConfig(
	cfg config.Config,
	configPath string,
	logger *log.Logger,
	releaseRuntime runtimestore.ActivatedRuntime,
	releaseRuntimeActivated bool,
) (bridgeruntime.ManagerConfig, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return bridgeruntime.ManagerConfig{}, err
	}

	appServerBin, _ := resolveAppServerBinary(cfg.Runtime.AppServerBin)
	if releaseRuntimeActivated && shouldPreferReleaseAppServer(cfg.Runtime.AppServerBin, config.DefaultConfig().Runtime.AppServerBin) {
		appServerBin = releaseRuntime.AppServerBin
	}
	if cfg.Runtime.Mode == config.RuntimeModeManaged && appServerBin == "" {
		return bridgeruntime.ManagerConfig{}, errors.New("no app-server executable available for managed mode")
	}
	runtimeCodexHome, err := ensureRuntimeCodexHome(cfg, configPath)
	if err != nil {
		return bridgeruntime.ManagerConfig{}, err
	}
	appServerArgs := bridgeManagedAppServerArgs(cfg.Runtime.AppServerArgs)

	return bridgeruntime.ManagerConfig{
		Mode:       cfg.Runtime.Mode,
		ListenHost: cfg.Runtime.ListenHost,
		ListenPort: cfg.Runtime.ListenPort,
		RemoteURL:  cfg.Runtime.RemoteURL,
		CodexHome:  runtimeCodexHome,
		AppServerCommand: bridgeruntime.Command{
			Path: appServerBin,
			Args: appServerArgs,
		},
		AutoRestart:  cfg.Runtime.AutoRestart,
		RestartDelay: time.Duration(cfg.Runtime.RestartDelayMS) * time.Millisecond,
		CWD:          workingDir,
		Logger:       logger,
	}, nil
}

func bridgeManagedAppServerArgs(base []string) []string {
	args := append([]string{}, base...)
	if hasAppServerFeatureDisable(args, "apps") {
		return args
	}

	// Bridge-hosted mobile sessions should not inherit desktop app/plugin integrations.
	// Those app calls depend on desktop-only MCP plumbing and can stall turns forever.
	args = append(args, "--disable", "apps")
	args = append(args, "-c", "features.apps=false")
	return args
}

func hasAppServerFeatureDisable(args []string, feature string) bool {
	normalizedFeature := strings.TrimSpace(strings.ToLower(feature))
	for index := 0; index < len(args); index++ {
		value := strings.TrimSpace(args[index])
		if value == "" {
			continue
		}
		switch {
		case value == "--disable":
			if index+1 < len(args) && strings.EqualFold(strings.TrimSpace(args[index+1]), normalizedFeature) {
				return true
			}
		case strings.HasPrefix(value, "--disable="):
			disabled := strings.TrimSpace(strings.TrimPrefix(value, "--disable="))
			if strings.EqualFold(disabled, normalizedFeature) {
				return true
			}
		case value == "-c":
			if index+1 < len(args) && isFeatureDisableConfigOverride(args[index+1], normalizedFeature) {
				return true
			}
		default:
			if isFeatureDisableConfigOverride(value, normalizedFeature) {
				return true
			}
		}
	}
	return false
}

func isFeatureDisableConfigOverride(arg string, feature string) bool {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" {
		return false
	}
	normalized := strings.ReplaceAll(strings.ToLower(trimmed), " ", "")
	return normalized == "features."+feature+"=false"
}

func ensureRuntimeCodexHome(cfg config.Config, configPath string) (string, error) {
	if cfg.Runtime.Mode != config.RuntimeModeManaged {
		return strings.TrimSpace(cfg.Runtime.CodexHome), nil
	}

	runtimeCodexHome := strings.TrimSpace(cfg.Runtime.CodexHome)
	if runtimeCodexHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		runtimeCodexHome = filepath.Join(homeDir, ".codex")
	}
	runtimeCodexHome, err := filepath.Abs(runtimeCodexHome)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(runtimeCodexHome, 0o755); err != nil {
		return "", err
	}
	return runtimeCodexHome, nil
}

func ensureBridgeID(configPath string) (string, error) {
	configDir := strings.TrimSpace(filepath.Dir(strings.TrimSpace(configPath)))
	if configDir == "" || configDir == "." {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".codex-bridge")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	identityPath := filepath.Join(configDir, "bridge-id")
	if content, err := os.ReadFile(identityPath); err == nil {
		if bridgeID := strings.TrimSpace(string(content)); bridgeID != "" {
			return bridgeID, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	bridgeID := "bridge_" + hex.EncodeToString(randomBytes)
	if err := os.WriteFile(identityPath, []byte(bridgeID+"\n"), 0o600); err != nil {
		return "", err
	}
	return bridgeID, nil
}

func buildBridgeConfig(
	cfg config.Config,
	configPath string,
	upstreamURL string,
	releaseRuntime runtimestore.ActivatedRuntime,
	releaseRuntimeActivated bool,
) (bridge.Config, error) {
	releaseRuntimeDesktopWebviewRoot := ""
	if releaseRuntimeActivated {
		releaseRuntimeDesktopWebviewRoot = releaseRuntime.DesktopWebviewRoot
	}

	desktopWebviewRoot, err := resolveDesktopWebviewRoot(cfg.Gateway.DesktopWebviewRoot, releaseRuntimeDesktopWebviewRoot)
	if err != nil {
		return bridge.Config{}, err
	}
	if desktopWebviewRoot == "" {
		desktopWebviewRoot, err = ensureBundledDesktopWebviewRoot(configPath)
		if err != nil {
			return bridge.Config{}, err
		}
	}
	if desktopWebviewRoot == "" {
		return bridge.Config{}, fmt.Errorf(
			"no desktop webview bundle available; provide --desktop-webview-root, place desktop-webview assets next to the bridge binary, or activate a packaged runtime",
		)
	}
	bridgeID, err := ensureBridgeID(configPath)
	if err != nil {
		return bridge.Config{}, err
	}

	return bridge.Config{
		BridgeID:           bridgeID,
		Host:               cfg.Gateway.Host,
		Port:               cfg.Gateway.Port,
		UpstreamURL:        upstreamURL,
		HealthEnabled:      true,
		HealthPath:         "/healthz",
		ReadyPath:          "/readyz",
		ProbeTimeout:       2 * time.Second,
		ProbeCacheTTL:      5 * time.Second,
		DesktopWebviewRoot: desktopWebviewRoot,
		UIPathPrefix:       cfg.Gateway.UIPathPrefix,
		MobileEnrollment: bridge.MobileEnrollmentConfig{
			ControlURL:        cfg.Exposure.Tailnet.MobileControlURL,
			Hostname:          cfg.Exposure.Tailnet.MobileHostname,
			OAuthClientID:     cfg.Exposure.Tailnet.MobileOAuthClientID,
			OAuthClientSecret: cfg.Exposure.Tailnet.MobileOAuthClientSecret,
			OAuthTailnet:      cfg.Exposure.Tailnet.MobileOAuthTailnet,
			OAuthTags:         append([]string{}, cfg.Exposure.Tailnet.MobileOAuthTags...),
		},
	}, nil
}

func buildExposureProvider(cfg config.Config, logger *log.Logger) (exposure.Provider, error) {
	switch cfg.Exposure.Mode {
	case config.ExposureModeNone:
		return exposurenoop.New(), nil
	case config.ExposureModeLocal:
		return exposurelocal.New(), nil
	case config.ExposureModeTunnel:
		return exposuretunnel.New(exposuretunnel.Config{
			SSHBinary:      cfg.Exposure.Tunnel.SSHBinary,
			SSHDestination: cfg.Exposure.Tunnel.SSHDestination,
			SSHPort:        cfg.Exposure.Tunnel.SSHPort,
			RemotePort:     cfg.Exposure.Tunnel.RemotePort,
			PublicHost:     cfg.Exposure.Tunnel.PublicHost,
			PublicPort:     cfg.Exposure.Tunnel.PublicPort,
			PublicScheme:   cfg.Exposure.Tunnel.PublicScheme,
			SSHArgs:        append([]string{}, cfg.Exposure.Tunnel.SSHArgs...),
			AutoRestart:    cfg.Exposure.Tunnel.AutoRestart,
			RestartDelay:   time.Duration(cfg.Exposure.Tunnel.RestartDelayMS) * time.Millisecond,
			Logger:         logger,
		}), nil
	case config.ExposureModeTailnet:
		return exposuretailnet.New(exposuretailnet.Config{
			Socket:          cfg.Exposure.Tailnet.Socket,
			Hostname:        cfg.Exposure.Tailnet.Hostname,
			AddressStrategy: cfg.Exposure.Tailnet.AddressStrategy,
			Logger:          logger,
		}), nil
	case config.ExposureModeLibp2p:
		return exposurelibp2p.New(exposurelibp2p.Config{
			ListenAddrs:     append([]string{}, cfg.Exposure.Libp2p.ListenAddrs...),
			BootstrapPeers:  append([]string{}, cfg.Exposure.Libp2p.BootstrapPeers...),
			PrivateKeyPath:  cfg.Exposure.Libp2p.PrivateKeyPath,
			EnableRelay:     cfg.Exposure.Libp2p.EnableRelay,
			EnableMDNS:      cfg.Exposure.Libp2p.EnableMDNS,
			ProxyListenPort: cfg.Exposure.Libp2p.ProxyListenPort,
			Logger:          logger,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported exposure mode %q", cfg.Exposure.Mode)
	}
}

func buildAuthorizer(cfg config.Config, configPath string) (auth.Authorizer, error) {
	switch cfg.Auth.Mode {
	case config.AuthModeNone:
		return auth.NewNoopAuthorizer(), nil
	case config.AuthModeDeviceToken:
		storePath := strings.TrimSpace(cfg.Auth.DeviceStorePath)
		if storePath == "" {
			storePath = filepath.Join(filepath.Dir(configPath), "auth-devices.json")
		}
		return auth.NewDeviceTokenAuthorizer(auth.DeviceTokenConfig{
			StorePath:       storePath,
			RequireApproval: cfg.Auth.RequireApproval,
			PairingCodeTTL:  time.Duration(cfg.Auth.PairingCodeTTLMS) * time.Millisecond,
		})
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", cfg.Auth.Mode)
	}
}

func ensureReleaseRuntime(dataDir string, logger *log.Logger, requireAppServer bool) (runtimestore.ActivatedRuntime, bool, error) {
	platform := runtimePlatform()
	pointerCandidates, err := releaseRuntimeCandidates("bridge-runtime-manifest-" + platform + ".json")
	if err != nil {
		return runtimestore.ActivatedRuntime{}, false, err
	}
	manifestCandidates, err := releaseRuntimeCandidates("runtime-manifest-" + platform + ".json")
	if err != nil {
		return runtimestore.ActivatedRuntime{}, false, err
	}
	archiveCandidates, err := releaseRuntimeCandidates("runtime-" + platform + ".tar.gz")
	if err != nil {
		return runtimestore.ActivatedRuntime{}, false, err
	}

	return runtimestore.EnsureRuntime(runtimestore.Options{
		DataDir:                   dataDir,
		PointerManifestCandidates: pointerCandidates,
		LocalManifestCandidates:   manifestCandidates,
		LocalArchiveCandidates:    archiveCandidates,
		RequireAppServer:          requireAppServer,
		Logger:                    logger,
	})
}

func shouldPreferReleaseAppServer(configured string, defaultValue string) bool {
	trimmed := strings.TrimSpace(configured)
	if trimmed == "" {
		return true
	}
	return trimmed == strings.TrimSpace(defaultValue)
}

func resolveDesktopWebviewRoot(configured string, releaseRuntimeRoot string) (string, error) {
	if root, ok, err := resolveExistingDirectory(configured); ok || err != nil {
		return root, err
	}
	if root, ok, err := resolveExistingDirectory(releaseRuntimeRoot); ok || err != nil {
		return root, err
	}

	candidates, err := desktopWebviewCandidates()
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if root, ok, err := resolveExistingDirectory(candidate); ok || err != nil {
			return root, err
		}
	}
	return "", nil
}

func resolveAppServerBinary(configured string) (string, error) {
	if preferred := preferredBundledAppServerBinary(); preferred != "" {
		if strings.TrimSpace(configured) == "" || strings.TrimSpace(configured) == "codex" {
			return preferred, nil
		}
	}

	if binary, ok, err := resolveExecutable(configured); ok || err != nil {
		return binary, err
	}

	candidates, err := appServerCandidates()
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if binary, ok, err := resolveExecutable(candidate); ok || err != nil {
			return binary, err
		}
	}
	return "", nil
}

func ensureBundledDesktopWebviewRoot(configPath string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", nil
	}

	asarPath, ok, err := resolveExistingFile(bundledDesktopWebviewAppAsarPath)
	if err != nil || !ok {
		return "", err
	}
	info, err := os.Stat(asarPath)
	if err != nil {
		return "", err
	}

	cacheRoot := filepath.Join(
		bridgeDataDir(configPath),
		"cache",
		"upstream-desktop-webview",
		fmt.Sprintf("%d-%d", info.Size(), info.ModTime().UTC().Unix()),
	)
	return extractBundledDesktopWebviewRoot(asarPath, cacheRoot)
}

func extractBundledDesktopWebviewRoot(asarPath string, cacheRoot string) (string, error) {
	webviewRoot := filepath.Join(strings.TrimSpace(cacheRoot), "webview")
	if root, ok, err := resolveExistingDirectory(webviewRoot); ok || err != nil {
		return root, err
	}

	npxPath, err := execLookPath("npx")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(cacheRoot), 0o755); err != nil {
		return "", err
	}

	tempRoot := cacheRoot + ".tmp"
	_ = os.RemoveAll(tempRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := execCommandContext(ctx, npxPath, "--yes", "asar", "extract", asarPath, tempRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		if ctx.Err() != nil {
			return "", fmt.Errorf("extract desktop webview from bundled Codex.app timed out: %w", ctx.Err())
		}
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return "", fmt.Errorf("extract desktop webview from bundled Codex.app: %w", err)
		}
		return "", fmt.Errorf("extract desktop webview from bundled Codex.app: %w: %s", err, detail)
	}

	_, ok, err := resolveExistingDirectory(filepath.Join(tempRoot, "webview"))
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}
	if !ok {
		_ = os.RemoveAll(tempRoot)
		return "", fmt.Errorf("bundled Codex.app archive does not contain webview/index.html")
	}

	_ = os.RemoveAll(cacheRoot)
	if err := os.Rename(tempRoot, cacheRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}
	return filepath.Join(cacheRoot, "webview"), nil
}

func preferredBundledAppServerBinary() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	info, err := os.Stat(macOSBundledCodexPath)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return ""
	}
	return macOSBundledCodexPath
}

func desktopWebviewCandidates() ([]string, error) {
	exeDir, err := executableDir()
	if err != nil {
		return nil, err
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	candidates := []string{
		filepath.Join(exeDir, "desktop-webview"),
		filepath.Join(exeDir, "resources", "desktop-webview"),
		filepath.Join(exeDir, "..", "Resources", "desktop-webview"),
		filepath.Join(workingDir, "desktop-webview"),
		filepath.Join(workingDir, "resources", "desktop-webview"),
	}
	candidates = append(candidates, ancestorAssetCandidates(workingDir, "assets", "desktop-webview")...)
	return uniqueStrings(candidates), nil
}

func releaseRuntimeCandidates(fileName string) ([]string, error) {
	exeDir, err := executableDir()
	if err != nil {
		return nil, err
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return uniqueStrings([]string{
		filepath.Join(exeDir, "runtime", fileName),
		filepath.Join(exeDir, "resources", "runtime", fileName),
		filepath.Join(exeDir, "..", "Resources", "runtime", fileName),
		filepath.Join(exeDir, fileName),
		filepath.Join(exeDir, "resources", fileName),
		filepath.Join(exeDir, "..", "Resources", fileName),
		filepath.Join(workingDir, "runtime", fileName),
		filepath.Join(workingDir, "resources", "runtime", fileName),
		filepath.Join(workingDir, fileName),
		filepath.Join(workingDir, "resources", fileName),
	}), nil
}

func appServerCandidates() ([]string, error) {
	exeDir, err := executableDir()
	if err != nil {
		return nil, err
	}

	candidates := []string{
		preferredBundledAppServerBinary(),
		filepath.Join(exeDir, "codex"),
		filepath.Join(exeDir, "resources", "codex"),
		filepath.Join(exeDir, "..", "Resources", "codex"),
		"codex",
	}
	return uniqueStrings(candidates), nil
}

func executableDir() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolvedPath, err := filepath.EvalSymlinks(executablePath); err == nil {
		executablePath = resolvedPath
	}
	return filepath.Dir(executablePath), nil
}

func runtimePlatform() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return runtime.GOOS + "-" + arch
}

func bridgeDataDir(configPath string) string {
	trimmed := strings.TrimSpace(configPath)
	if trimmed == "" {
		trimmed = config.DefaultConfigPath()
	}
	return filepath.Dir(trimmed)
}

func resolveExistingFile(path string) (string, bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false, nil
	}
	resolved, err := filepath.Abs(trimmed)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if info.IsDir() {
		return "", false, nil
	}
	return resolved, true, nil
}

func resolveExistingDirectory(path string) (string, bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false, nil
	}
	resolved, err := filepath.Abs(trimmed)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.IsDir() {
		return "", false, nil
	}
	indexPath := filepath.Join(resolved, "index.html")
	indexInfo, err := os.Stat(indexPath)
	if err != nil || indexInfo.IsDir() {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		return "", false, nil
	}
	return resolved, true, nil
}

func resolveExecutable(path string) (string, bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false, nil
	}

	if filepath.IsAbs(trimmed) || strings.ContainsRune(trimmed, os.PathSeparator) {
		resolved, err := filepath.Abs(trimmed)
		if err != nil {
			return "", false, err
		}
		return validateExecutable(resolved)
	}

	resolved, err := exec.LookPath(trimmed)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return validateExecutable(resolved)
}

func validateExecutable(path string) (string, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return "", false, nil
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}
	return resolved, true, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func ancestorAssetCandidates(startDir string, relativeParts ...string) []string {
	current := filepath.Clean(startDir)
	var result []string

	for {
		parts := append([]string{current}, relativeParts...)
		result = append(result, filepath.Join(parts...))
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return result
}
