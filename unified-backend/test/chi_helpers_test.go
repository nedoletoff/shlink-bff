package test

import (
	"github.com/go-chi/chi/v5"
)

func chiContext() *chi.Context {
	return chi.NewRouteContext()
}
