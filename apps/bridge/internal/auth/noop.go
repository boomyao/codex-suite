package auth

import "net/http"

type NoopAuthorizer struct{}

func NewNoopAuthorizer() Authorizer {
	return NoopAuthorizer{}
}

func (NoopAuthorizer) State() State {
	return State{Mode: "none", RequireApproval: false}
}

func (NoopAuthorizer) AuthorizeRequest(*http.Request) error {
	return nil
}

func (NoopAuthorizer) DescribeRequest(*http.Request) SessionInfo {
	return SessionInfo{Authorized: true, Reason: "authorized"}
}

func (NoopAuthorizer) HandleHTTP(http.ResponseWriter, *http.Request) bool {
	return false
}
