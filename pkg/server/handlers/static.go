package handlers

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

		_, err = w.Write(bs)
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
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

		_, err = w.Write(ff.Bytes)
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
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

func BuildContentTypeIconHandler(opts *Options) (handler func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		lookup := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/icons/content-types/"), ".svg")

		var key string
		switch lookup {
		case "image/jpg", "image/jpeg":
			key = "jpg"
		case "image/png":
			key = "png"
		case "image/gif":
			key = "gif"
		case "image/heic":
			key = "heic"
		case "video/mp4":
			key = "mp4"
		case "application/pdf":
			key = "pdf"
		case "custom/folder":
			key = "folder"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			key = "docx"
		case "text/calendar":
			key = "ics"
		case "video/quicktime":
			key = "mov"
		case "text/csv":
			key = "csv"
		default:
			key = "blank"
		}

		iconReader, err := staticContent.Open(fmt.Sprintf("static/icons/content-types/%s.svg", key))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte("failed to open icon"))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}
		defer iconReader.Close()

		iconBytes, err := io.ReadAll(iconReader)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte("failed to read icon"))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		etag := utils.CRC32Hash(iconBytes)

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(iconBytes)))
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Expires", time.Now().AddDate(10, 0, 0).Format(http.TimeFormat))

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		_, err = io.Copy(w, bytes.NewReader(iconBytes))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte("failed to copy icon data"))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}
			return
		}
	}
}

func BuildCSSHandler(opts *Options) (string, func(http.ResponseWriter, *http.Request), error) {
	sourceFileOrder := []string{
		"tachyons.css",
		"styles.css",
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

		_, err := w.Write(out.Bytes())
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
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

		_, err := w.Write(out.Bytes())
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
	}, nil
}
