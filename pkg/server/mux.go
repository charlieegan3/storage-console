package server

import (
	"fmt"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/handlers"
	"github.com/charlieegan3/storage-console/pkg/middlewares"
)

func newMux(opts *handlers.Options) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	stylesEtag, stylesHandler, err := handlers.BuildCSSHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build styles handler: %s", err)
	}

	scriptETag, scriptHandler, err := handlers.BuildJSHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build script handler: %s", err)
	}

	opts.EtagStyles = stylesEtag
	opts.EtagScript = scriptETag

	indexHandler, err := handlers.BuildIndexHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build index handler: %s", err)
	}

	mux.Handle(
		"/script.js",
		middlewares.BuildAuth(http.HandlerFunc(scriptHandler), opts),
	)
	mux.Handle(
		"/styles.css",
		middlewares.BuildAuth(http.HandlerFunc(stylesHandler), opts),
	)
	mux.Handle(
		"/",
		middlewares.BuildAuth(http.HandlerFunc(indexHandler), opts),
	)

	return mux, nil
}