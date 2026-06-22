// Package openrouter is a stub provider — see openai/ for rationale.
// Worth implementing first when real demand lands since OpenRouter
// exposes every model behind one API, which simplifies BYOK ergonomics.
package openrouter

import (
	"context"

	"alethea/api/services/ai/types"
)

type Provider struct {
	apiKey string
	model  string
}

func NewProvider(cfg types.Config) *Provider {
	model := cfg.Model
	if model == "" {
		model = "anthropic/claude-3.5-sonnet"
	}
	return &Provider{apiKey: cfg.APIKey, model: model}
}

func (p *Provider) Name() string                 { return "openrouter" }
func (p *Provider) ModelID() string              { return p.model }
func (p *Provider) SupportsVision() bool         { return true }
func (p *Provider) Chat(_ string) (string, error) {
	return "", types.ErrNotImplemented
}
func (p *Provider) ChatWithSystem(_, _ string) (string, error) {
	return "", types.ErrNotImplemented
}
func (p *Provider) ChatWithSystemCtx(_ context.Context, _, _ string, _ types.CompleteOpts) (string, types.Usage, error) {
	return "", types.Usage{}, types.ErrNotImplemented
}
func (p *Provider) CompleteJSON(_ context.Context, _, _ string, _ types.CompleteOpts) ([]byte, types.Usage, error) {
	return nil, types.Usage{}, types.ErrNotImplemented
}
func (p *Provider) AnalyzeImage(_, _ string) (string, error) {
	return "", types.ErrNotImplemented
}
