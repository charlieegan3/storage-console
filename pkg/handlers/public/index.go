package public

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/handlers"
	"github.com/charlieegan3/storage-console/pkg/stores"
)

func BuildIndexHandler(
	sessionDB *stores.SessionDB,
	userDB *stores.UserDB,
	db *sql.DB,
	opts *handlers.Options,
) (func(http.ResponseWriter, *http.Request), error) {
	tmpl, err := template.ParseFS(
		handlers.Templates,
		"templates/index.html",
		"templates/base.html",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %s", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.ExecuteTemplate(w, "base", struct {
			Opts *handlers.Options
		}{Opts: opts})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}, nil
}
