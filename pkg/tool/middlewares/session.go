package middlewares

import (
	"context"
	"net/http"

	"github.com/charlieegan3/curry-club/pkg/tool/stores"
)

func BuildSessionMiddleware(sessionDB *stores.SessionDB, userDB *stores.UserDB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err == nil {
				session, err := sessionDB.GetSession(cookie.Value)
				if err != nil {
					http.Redirect(w, r, "/logout", http.StatusFound)
					return
				}

				user, err := userDB.GetUserByID(string(session.UserID))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), "userID", user.ID))
				r = r.WithContext(context.WithValue(r.Context(), "userName", user.Username))
			}

			next.ServeHTTP(w, r)
		})
	}
}
