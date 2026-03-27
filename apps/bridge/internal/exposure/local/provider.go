package local

import (
	"context"
	"sync"

	"github.com/boomyao/codex-bridge/internal/exposure"
)

type Provider struct {
	mu      sync.Mutex
	session *exposure.Session
}

func New() exposure.Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "local"
}

func (p *Provider) Start(_ context.Context, target exposure.Target) (*exposure.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.session = &exposure.Session{
		ID:            "local",
		Provider:      "local",
		ReachableHTTP: target.GatewayHTTPURL,
		ReachableWS:   target.GatewayWebSocketURL,
		Status:        "ready",
	}
	copy := *p.session
	return &copy, nil
}

func (p *Provider) Status(_ context.Context) (*exposure.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session == nil {
		return &exposure.Session{
			ID:       "local",
			Provider: "local",
			Status:   "idle",
		}, nil
	}
	copy := *p.session
	return &copy, nil
}

func (p *Provider) Stop(_ context.Context, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.session = nil
	return nil
}
