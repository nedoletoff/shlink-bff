package test

import (
	"github.com/go-chi/chi/v5"
)

// chiRouteContextKey is the same unexported type chi uses internally.
// We re-declare it to inject URL params in tests without running a full router.
type chiRouteContextKey struct{}

func chiContext() *chi.Context {
	return chi.NewRouteContext()
}
