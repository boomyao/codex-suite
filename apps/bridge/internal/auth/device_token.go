package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DeviceTokenConfig struct {
	StorePath       string
	RequireApproval bool
	PairingCodeTTL  time.Duration
}

type DeviceTokenAuthorizer struct {
	storePath       string
	requireApproval bool
	pairingCodeTTL  time.Duration

	mu          sync.Mutex
	devices     map[string]*deviceRecord
	pairingCode *pairingCode
}

type pairingCode struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type deviceRecord struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"tokenHash"`
	Approved   bool       `json:"approved"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type deviceStoreFile struct {
	Devices []*deviceRecord `json:"devices"`
}

type publicDevice struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Approved   bool       `json:"approved"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

func NewDeviceTokenAuthorizer(config DeviceTokenConfig) (Authorizer, error) {
	storePath := strings.TrimSpace(config.StorePath)
	if storePath == "" {
		return nil, errors.New("device token store path is required")
	}
	if config.PairingCodeTTL <= 0 {
		config.PairingCodeTTL = 5 * time.Minute
	}

	authorizer := &DeviceTokenAuthorizer{
		storePath:       storePath,
		requireApproval: config.RequireApproval,
		pairingCodeTTL:  config.PairingCodeTTL,
		devices:         map[string]*deviceRecord{},
	}
	if err := authorizer.load(); err != nil {
		return nil, err
	}
	return authorizer, nil
}

func (a *DeviceTokenAuthorizer) State() State {
	return State{Mode: "device-token", RequireApproval: a.requireApproval}
}

func (a *DeviceTokenAuthorizer) AuthorizeRequest(r *http.Request) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.describeRequestLocked(r)
	if !session.Authorized {
		return ErrUnauthorized
	}

	device := a.devices[session.DeviceID]
	if device == nil {
		return ErrUnauthorized
	}
	device.LastSeenAt = time.Now().UTC()
	if err := a.persistLocked(); err != nil {
		return err
	}
	return nil
}

func (a *DeviceTokenAuthorizer) DescribeRequest(r *http.Request) SessionInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.describeRequestLocked(r)
}

func (a *DeviceTokenAuthorizer) HandleHTTP(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.URL.Path == "/auth/pair/start":
		if r.Method != http.MethodPost {
			sendJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Method Not Allowed"})
			return true
		}
		if !isLoopbackRequest(r) {
			sendJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "Pair start requires local access"})
			return true
		}
		a.handlePairStart(w)
		return true
	case r.URL.Path == "/auth/pair/complete":
		if r.Method != http.MethodPost {
			sendJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Method Not Allowed"})
			return true
		}
		a.handlePairComplete(w, r)
		return true
	case r.URL.Path == "/auth/devices":
		if r.Method != http.MethodGet {
			sendJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Method Not Allowed"})
			return true
		}
		if !isLoopbackRequest(r) {
			sendJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "Device management requires local access"})
			return true
		}
		a.handleListDevices(w)
		return true
	case strings.HasPrefix(r.URL.Path, "/auth/devices/"):
		if !isLoopbackRequest(r) {
			sendJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "Device management requires local access"})
			return true
		}
		a.handleDeviceAction(w, r)
		return true
	default:
		return false
	}
}

func (a *DeviceTokenAuthorizer) handlePairStart(w http.ResponseWriter) {
	code, err := randomDigits(8)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	expiresAt := time.Now().UTC().Add(a.pairingCodeTTL)
	a.mu.Lock()
	a.pairingCode = &pairingCode{
		Code:      code,
		ExpiresAt: expiresAt,
	}
	a.mu.Unlock()

	sendJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"code":            code,
		"expiresAt":       expiresAt,
		"requireApproval": a.requireApproval,
	})
}

func (a *DeviceTokenAuthorizer) handlePairComplete(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Code       string `json:"code"`
		DeviceName string `json:"deviceName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid JSON body"})
		return
	}

	payload.Code = strings.TrimSpace(payload.Code)
	payload.DeviceName = strings.TrimSpace(payload.DeviceName)
	if payload.Code == "" || payload.DeviceName == "" {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "code and deviceName are required"})
		return
	}

	token, err := randomToken(24)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	deviceID, err := randomToken(12)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	now := time.Now().UTC()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.pairingCode == nil || now.After(a.pairingCode.ExpiresAt) || a.pairingCode.Code != payload.Code {
		sendJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Invalid or expired pairing code"})
		return
	}

	record := &deviceRecord{
		ID:         "dev_" + deviceID,
		Name:       payload.DeviceName,
		TokenHash:  hashToken(token),
		Approved:   !a.requireApproval,
		CreatedAt:  now,
		LastSeenAt: now,
	}
	a.devices[record.ID] = record
	a.pairingCode = nil
	if err := a.persistLocked(); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"deviceId":    record.ID,
		"accessToken": token,
		"approved":    record.Approved,
	})
}

func (a *DeviceTokenAuthorizer) handleListDevices(w http.ResponseWriter) {
	a.mu.Lock()
	defer a.mu.Unlock()

	devices := make([]publicDevice, 0, len(a.devices))
	for _, device := range a.devices {
		devices = append(devices, publicDevice{
			ID:         device.ID,
			Name:       device.Name,
			Approved:   device.Approved,
			CreatedAt:  device.CreatedAt,
			LastSeenAt: device.LastSeenAt,
			RevokedAt:  device.RevokedAt,
		})
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"devices": devices,
	})
}

func (a *DeviceTokenAuthorizer) handleDeviceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Method Not Allowed"})
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/auth/devices/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		sendJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "Not Found"})
		return
	}

	deviceID := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])

	a.mu.Lock()
	defer a.mu.Unlock()

	device := a.devices[deviceID]
	if device == nil {
		sendJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "Unknown device"})
		return
	}

	now := time.Now().UTC()
	switch action {
	case "approve":
		device.Approved = true
	case "revoke":
		device.RevokedAt = &now
	default:
		sendJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "Unknown action"})
		return
	}

	if err := a.persistLocked(); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *DeviceTokenAuthorizer) load() error {
	raw, err := os.ReadFile(a.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var store deviceStoreFile
	if err := json.Unmarshal(raw, &store); err != nil {
		return err
	}

	for _, device := range store.Devices {
		if device == nil || strings.TrimSpace(device.ID) == "" {
			continue
		}
		a.devices[device.ID] = device
	}
	return nil
}

func (a *DeviceTokenAuthorizer) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(a.storePath), 0o755); err != nil {
		return err
	}

	store := deviceStoreFile{
		Devices: make([]*deviceRecord, 0, len(a.devices)),
	}
	for _, device := range a.devices {
		copy := *device
		store.Devices = append(store.Devices, &copy)
	}

	body, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(a.storePath, body, 0o600)
}

func (a *DeviceTokenAuthorizer) findByTokenHash(hash string) *deviceRecord {
	for _, device := range a.devices {
		if device.TokenHash == hash {
			return device
		}
	}
	return nil
}

func (a *DeviceTokenAuthorizer) describeRequestLocked(r *http.Request) SessionInfo {
	token := extractBearerToken(r)
	if token == "" {
		return SessionInfo{Authorized: false, Reason: "missing_token"}
	}

	device := a.findByTokenHash(hashToken(token))
	if device == nil {
		return SessionInfo{Authorized: false, Reason: "unknown_token"}
	}

	session := SessionInfo{
		Authorized: false,
		Reason:     "authorized",
		DeviceID:   device.ID,
		DeviceName: device.Name,
	}
	if device.RevokedAt != nil {
		session.Reason = "revoked"
		return session
	}
	if !device.Approved {
		session.Reason = "pending_approval"
		return session
	}
	session.Authorized = true
	return session
}

func extractBearerToken(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Codex-Bridge-Token")); value != "" {
		return value
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	}
	return ""
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(numBytes int) (string, error) {
	buffer := make([]byte, numBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func randomDigits(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid pairing code length %d", length)
	}
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	result := make([]byte, length)
	for index, value := range buffer {
		result[index] = byte('0' + (value % 10))
	}
	return string(result), nil
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sendJSON(w http.ResponseWriter, statusCode int, payload any) {
	body, _ := json.MarshalIndent(payload, "", "  ")
	body = append(body, '\n')
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}
