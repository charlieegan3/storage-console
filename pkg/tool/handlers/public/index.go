package public

import (
	"database/sql"
	"net/http"

	"github.com/doug-martin/goqu/v9"
	"github.com/foolin/goview"

	"github.com/charlieegan3/curry-club/pkg/tool/handlers/status"
	"github.com/charlieegan3/curry-club/pkg/tool/types"
	"github.com/charlieegan3/curry-club/pkg/tool/views"
)

func BuildIndexHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		var block types.Block
		found, err := goquDB.From("curry_club.blocks").As("blocks").
			Where(goqu.Ex{"key": "index"}).
			Select("blocks.*").
			Limit(1).
			ScanStruct(&block)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if !found {
			status.NotFound(w, r)
			return
		}

		err = views.Engine.Render(
			w,
			http.StatusOK,
			"public/index",
			goview.M{
				"block_content": block.Content,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}
