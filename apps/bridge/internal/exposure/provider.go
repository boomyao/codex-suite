package exposure

import "context"

type Target struct {
	AppServerWebSocketURL string
	GatewayHTTPURL        string
	GatewayWebSocketURL   string
}

type Session struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	ReachableHTTP string `json:"reachableHttp"`
	ReachableWS   string `json:"reachableWs"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

type Provider interface {
	Name() string
	Start(ctx context.Context, target Target) (*Session, error)
	Status(ctx context.Context) (*Session, error)
	Stop(ctx context.Context, id string) error
}
