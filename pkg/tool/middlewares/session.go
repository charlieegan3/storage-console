package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/charlieegan3/curry-club/pkg/tool/handlers/public"
	"github.com/charlieegan3/curry-club/pkg/tool/stores"
)

func BuildSessionMiddleware(sessionDB *stores.SessionDB, userDB *stores.UserDB) func(http.Handler) http.Handler {
	logoutHandler := public.BuildLogoutUserHandler(sessionDB)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err == nil {
				session, err := sessionDB.GetSession(cookie.Value)
				if err != nil {
					logoutHandler(w, r)
					return
				}

				user, err := userDB.GetUserByID(string(session.UserID))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}

				// user did not complete registration
				if len(user.Credentials) == 0 &&
					!strings.HasPrefix(r.URL.Path, "/register/finish") {
					fmt.Println(r.URL.Path)
					err := userDB.DeleteUserByID(user.ID)
					if err != nil {
						log.Printf("error deleting user: %s", err.Error())
					}
					logoutHandler(w, r)
					return
				}

				r = r.WithContext(context.WithValue(r.Context(), "userID", user.ID))
				r = r.WithContext(context.WithValue(r.Context(), "userName", user.Username))
			}

			next.ServeHTTP(w, r)
		})
	}
}
