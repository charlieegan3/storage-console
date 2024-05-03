package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

func BuildIndexHandler(opts *Options) (func(http.ResponseWriter, *http.Request), error) {
	tmpl, err := template.ParseFS(
		Templates,
		"templates/index.html",
		"templates/base.html",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %s", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer([]byte{})

		err := tmpl.ExecuteTemplate(buf, "base", struct {
			Opts *Options
		}{
			Opts: opts,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}
			return
		}

		_, err = io.Copy(w, buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}
			return
		}
	}, nil
}
