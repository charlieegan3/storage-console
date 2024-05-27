package server

import (
	"fmt"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/server/handlers"
	"github.com/charlieegan3/storage-console/pkg/server/handlers/browse"
	"github.com/charlieegan3/storage-console/pkg/server/middlewares"
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

	browseHandler, err := browse.BuildHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build browse handler: %s", err)
	}

	mux.Handle(
		"/b/",
		middlewares.BuildAuth(http.HandlerFunc(browseHandler), opts),
	)

	mux.Handle(
		"/icons/content-types/",
		middlewares.BuildAuth(http.HandlerFunc(handlers.BuildContentTypeIconHandler(opts)), opts),
	)

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
