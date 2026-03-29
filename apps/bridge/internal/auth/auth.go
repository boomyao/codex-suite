package auth

import (
	"errors"
	"net/http"
	"time"
)

type State struct {
	Mode            string `json:"mode"`
	RequireApproval bool   `json:"requireApproval"`
}

type SessionInfo struct {
	Authorized bool   `json:"authorized"`
	Reason     string `json:"reason,omitempty"`
	DeviceID   string `json:"deviceId,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
}

type PairingInfo struct {
	Code            string    `json:"code"`
	ExpiresAt       time.Time `json:"expiresAt"`
	RequireApproval bool      `json:"requireApproval"`
}

type Authorizer interface {
	State() State
	AuthorizeRequest(r *http.Request) error
	DescribeRequest(r *http.Request) SessionInfo
	HandleHTTP(w http.ResponseWriter, r *http.Request) bool
}

var ErrUnauthorized = errors.New("unauthorized request")
