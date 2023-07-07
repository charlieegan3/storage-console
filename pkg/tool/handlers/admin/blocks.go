package admin

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/doug-martin/goqu/v9"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"

	"github.com/charlieegan3/curry-club/pkg/tool/types"
	"github.com/charlieegan3/curry-club/pkg/tool/views"
)

func BuildBlockIndexHandler(db *sql.DB, adminPath string) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		var blocks []types.Block
		err = goquDB.From("curry_club.blocks").Order(goqu.I("key").Asc()).ScanStructs(&blocks)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = views.Engine.Render(
			w,
			http.StatusOK,
			"admin/blocks/index",
			goview.M{
				"blocks":     blocks,
				"admin_path": adminPath,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func BuildBlockCreateHandler(db *sql.DB, adminPath string) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {

		key := r.FormValue("key")
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("No key provided"))
			return
		}

		var err error
		_, err = goquDB.Insert("curry_club.blocks").Rows(
			goqu.Record{
				"key":     r.FormValue("key"),
				"content": r.FormValue("content"),
			},
		).Returning("key").Executor().Exec()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		http.Redirect(w, r, fmt.Sprintf("%s/blocks", adminPath), http.StatusFound)
	}
}

func BuildBlockUpdateHandler(db *sql.DB, adminPath string) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		key, ok := mux.Vars(r)["blockKey"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("No key provided"))
			return
		}

		if r.FormValue("_method") == "DELETE" {
			_, err = goquDB.Delete("curry_club.blocks").
				Where(goqu.C("key").Eq(key)).
				Executor().Exec()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			http.Redirect(w, r, fmt.Sprintf("%s/blocks", adminPath), http.StatusFound)
		}

		_, err = goquDB.Update("curry_club.blocks").
			Set(goqu.Record{
				"content": r.FormValue("content"),
			}).
			Where(goqu.C("key").Eq(key)).
			Executor().Exec()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		http.Redirect(w, r, fmt.Sprintf("%s/blocks", adminPath), http.StatusFound)
	}
}
