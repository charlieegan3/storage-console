package middlewares

import (
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

func BuildAuth(h http.Handler, opts *handlers.Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.DevMode {
			h.ServeHTTP(w, r)
			return
		}

		_, err := w.Write([]byte("TODO"))
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}
