package noop

import (
	"context"

	"github.com/boomyao/codex-bridge/internal/exposure"
)

type Provider struct{}

func New() exposure.Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "none"
}

func (Provider) Start(context.Context, exposure.Target) (*exposure.Session, error) {
	return &exposure.Session{
		ID:       "noop",
		Provider: "none",
		Status:   "disabled",
	}, nil
}

func (Provider) Status(context.Context) (*exposure.Session, error) {
	return &exposure.Session{
		ID:       "noop",
		Provider: "none",
		Status:   "disabled",
	}, nil
}

func (Provider) Stop(context.Context, string) error {
	return nil
}
