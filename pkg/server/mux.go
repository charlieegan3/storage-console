package server

import (
	"fmt"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/handlers"
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

	mux.HandleFunc("/script.js", scriptHandler)
	mux.HandleFunc("/styles.css", stylesHandler)
	mux.Handle("/", http.HandlerFunc(indexHandler))

	return mux, nil
}
