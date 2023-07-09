package public

import (
	"database/sql"
	"encoding/binary"
	"net/http"
	"strconv"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/foolin/goview"

	"github.com/charlieegan3/curry-club/pkg/tool/handlers/status"
	"github.com/charlieegan3/curry-club/pkg/tool/stores"
	"github.com/charlieegan3/curry-club/pkg/tool/types"
	"github.com/charlieegan3/curry-club/pkg/tool/views"
)

func BuildIndexHandler(sessionDB *stores.SessionDB, userDB *stores.UserDB, db *sql.DB) func(http.ResponseWriter, *http.Request) {
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

		var loggedIn bool
		var userID uint64
		var userName string
		cookie, err := r.Cookie("authentication")
		if err == nil {
			sessionID, err := strconv.Atoi(cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			session, err := sessionDB.GetSession(uint64(sessionID))
			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:    "authentication",
					Value:   "",
					Path:    "/",
					Expires: time.Unix(0, 0),
				})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			loggedIn = true
			var n int
			userID, n = binary.Uvarint(session.UserID)
			if n <= 0 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			user, err := userDB.GetUserByID(userID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			userName = user.Name
		}

		err = views.Engine.Render(
			w,
			http.StatusOK,
			"public/index",
			goview.M{
				"block_content": block.Content,
				"logged_in":     loggedIn,
				"user_id":       userID,
				"user_name":     userName,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}
