// Package anthropic is a stub provider — see openai/ for rationale.
package anthropic

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
		model = "claude-opus-4-7"
	}
	return &Provider{apiKey: cfg.APIKey, model: model}
}

func (p *Provider) Name() string                 { return "anthropic" }
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
