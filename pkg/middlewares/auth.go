package middlewares

import (
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/handlers"
)

func BuildAuth(h http.Handler, opts *handlers.Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.DevMode {
			h.ServeHTTP(w, r)
		}

		w.Write([]byte("TODO"))
		w.WriteHeader(http.StatusUnauthorized)
	})
}
