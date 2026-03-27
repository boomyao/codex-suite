package auth

import (
	"errors"
	"net/http"
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

type Authorizer interface {
	State() State
	AuthorizeRequest(r *http.Request) error
	DescribeRequest(r *http.Request) SessionInfo
	HandleHTTP(w http.ResponseWriter, r *http.Request) bool
}

var ErrUnauthorized = errors.New("unauthorized request")
