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
			Opts     *Options
			LoggedIn bool
		}{
			Opts:     opts,
			LoggedIn: r.Context().Value("userID") != nil,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		_, err = io.Copy(w, buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

	}, nil
}
