package lyrics

import (
	"fmt"
)

// Resolver tries multiple providers in order.
type Resolver struct {
	providers []Provider
}

// NewResolver creates a resolver with the given providers.
// Providers are tried in the order they are given.
func NewResolver(providers ...Provider) *Resolver {
	return &Resolver{providers: append([]Provider(nil), providers...)}
}

// Find returns the first available lyrics from configured providers.
func (r *Resolver) Find(track TrackInfo) (Lyrics, error) {
	var errs []error
	for _, provider := range r.providers {
		lyrics, err := provider.find(track)
		if err == nil {
			return lyrics, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", provider.name(), err))
	}
	return Lyrics{}, &noLyricsFoundError{providerErrs: errs}
}

type noLyricsFoundError struct {
	providerErrs []error
}

func (e *noLyricsFoundError) Error() string {
	return "no lyrics found"
}

func (e *noLyricsFoundError) Unwrap() []error {
	return e.providerErrs
}
