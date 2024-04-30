package handlers

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"

	"github.com/charlieegan3/storage-console/pkg/utils"
)

//go:embed static/*
var staticContent embed.FS

func BuildFaviconHandler(opts *Options) (handler func(http.ResponseWriter, *http.Request)) {
	bs, err := staticContent.ReadFile("static/favicon.ico")
	if err != nil {
		panic(err)
	}

	etag := utils.CRC32Hash(bs)

	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", etag)
		if !opts.DevMode {
			utils.SetCacheControl(w, "public, max-age=3600")
		}

		w.Write(bs)
	}
}

type fontFile struct {
	Name  string
	ETag  string
	Bytes []byte
}

func BuildFontHandler(opts *Options) (handler func(http.ResponseWriter, *http.Request)) {
	files, err := staticContent.ReadDir("static/fonts")
	if err != nil {
		panic(err)
	}

	fontFiles := make(map[string]fontFile)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		bs, err := staticContent.ReadFile("static/fonts/" + f.Name())
		if err != nil {
			panic(err)
		}

		etag := utils.CRC32Hash(bs)

		fontFiles[f.Name()] = fontFile{
			Name:  f.Name(),
			ETag:  etag,
			Bytes: bs,
		}
	}

	return func(w http.ResponseWriter, req *http.Request) {
		fontPath := strings.TrimPrefix(req.URL.Path, "/fonts/")

		ff, ok := fontFiles[fontPath]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if req.Header.Get("If-None-Match") == ff.ETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		if strings.HasSuffix(ff.Name, ".woff2") {
			w.Header().Set("Content-Type", "application/font-woff2")
		} else if strings.HasSuffix(ff.Name, ".woff") {
			w.Header().Set("Content-Type", "application/font-woff")
		} else if strings.HasSuffix(ff.Name, ".ttf") {
			w.Header().Set("Content-Type", "font/ttf")
		}

		w.Header().Set("ETag", ff.ETag)
		if !opts.DevMode {
			utils.SetCacheControl(w, "public, max-age=3600")
		}

		w.Write(ff.Bytes)
	}
}

func BuildStaticHandler(opts *Options) (handler func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, req *http.Request) {
		if !opts.DevMode {
			utils.SetCacheControl(w, "public, max-age=3600")
		}

		rootedReq := http.Request{
			URL: &url.URL{
				Path: strings.TrimPrefix(req.URL.Path, "/static/"),
			},
		}

		http.FileServer(http.FS(staticContent)).ServeHTTP(w, &rootedReq)
	}
}

func BuildCSSHandler(opts *Options) (string, func(http.ResponseWriter, *http.Request), error) {
	sourceFileOrder := []string{
		"tachyons.css",
	}

	var bs []byte

	for _, f := range sourceFileOrder {
		fileBytes, err := staticContent.ReadFile("static/css/" + f)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate css: %s", err)
		}

		bs = append(bs, fileBytes...)
		bs = append(bs, []byte("\n")...)
	}

	in := bytes.NewBuffer(bs)
	out := bytes.NewBuffer([]byte{})
	if opts.DevMode {
		out = in
	} else {
		m := minify.New()
		m.AddFunc("application/css", css.Minify)

		if err := m.Minify("application/css", out, in); err != nil {
			return "", nil, fmt.Errorf("failed to generate css: %s", err)
		}
	}

	etag := utils.CRC32Hash(out.Bytes())

	return etag, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "text/css")
		w.Header().Set("ETag", etag)
		if !opts.DevMode {
			utils.SetCacheControl(w, "public, max-age=31622400")
		}

		w.Write(out.Bytes())
	}, nil
}

func BuildJSHandler(opts *Options) (string, func(http.ResponseWriter, *http.Request), error) {
	sourceFileOrder := []string{
		"jquery.js",
		"htmx.js",
		"htmx-preload.js",
		"script.js",
	}

	var bs []byte

	for _, f := range sourceFileOrder {
		fileBytes, err := staticContent.ReadFile("static/js/" + f)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate css: %s", err)
		}

		bs = append(bs, fileBytes...)
		bs = append(bs, []byte("\n")...)
	}

	in := bytes.NewBuffer(bs)
	out := bytes.NewBuffer([]byte{})

	if opts.DevMode {
		out = in
	} else {
		m := minify.New()
		m.AddFunc("application/javascript", js.Minify)

		if err := m.Minify("application/javascript", out, in); err != nil {
			return "", nil, fmt.Errorf("failed to generate js: %s", err)
		}
	}

	etag := utils.CRC32Hash(out.Bytes())

	return etag, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("ETag", etag)
		if !opts.DevMode {
			utils.SetCacheControl(w, "public, max-age=31622400")
		}

		w.Write(out.Bytes())
	}, nil
}
