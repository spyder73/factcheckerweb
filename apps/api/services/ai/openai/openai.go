// Package openai is a stub provider. Implements the interface skeleton so
// the factory can pick it up via OPENAI_API_KEY, but every method returns
// ErrNotImplemented. Replace with a real Chat Completions client when
// there's actual demand — Mistral covers the in-house path for now.
package openai

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
		model = "gpt-4o"
	}
	return &Provider{apiKey: cfg.APIKey, model: model}
}

func (p *Provider) Name() string              { return "openai" }
func (p *Provider) ModelID() string           { return p.model }
func (p *Provider) SupportsVision() bool      { return true }
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
