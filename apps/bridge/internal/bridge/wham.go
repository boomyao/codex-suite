package bridge

import "net/http"

func (b *Bridge) handleWhamStub(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/wham/accounts/check":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
			return true
		}
		sendJSON(w, http.StatusOK, map[string]any{
			"account_ordering": []string{},
			"accounts":         map[string]any{},
		})
		return true
	case "/wham/tasks/list":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
			return true
		}
		sendJSON(w, http.StatusOK, map[string]any{
			"items": []any{},
		})
		return true
	case "/wham/environments":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
			return true
		}
		sendJSON(w, http.StatusOK, []any{})
		return true
	case "/wham/usage":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
			return true
		}
		sendJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		return false
	}
}
