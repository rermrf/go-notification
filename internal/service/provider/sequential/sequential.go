package sequential

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/service/provider"
)

var (
	_ provider.Selector        = (*selector)(nil)
	_ provider.SelectorBuilder = (*SelectorBuilder)(nil)
)

// selector 供应商顺序选择器
type selector struct {
	idx       int
	providers []provider.Provider
}

func (s selector) Next(_ context.Context, _ domain.Notification) (provider.Provider, error) {
	if len(s.providers) == s.idx {
		return nil, fmt.Errorf("%w", errs.ErrNoAvailableProvider)
	}

	p := s.providers[s.idx]
	s.idx++
	return p, nil
}

type SelectorBuilder struct {
	providers []provider.Provider
}

func NewSelectorBuilder(providers []provider.Provider) *SelectorBuilder {
	return &SelectorBuilder{providers: providers}
}

func (s SelectorBuilder) Build() (provider.Selector, error) {
	return &selector{
		providers: s.providers,
	}, nil
}
