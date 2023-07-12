package middlewares

import (
	"context"
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
				session, authd, err := sessionDB.GetSession(cookie.Value)
				if err != nil {
					log.Printf("error getting session: %s", err.Error())
					logoutHandler(w, r)
					return
				}

				user, err := userDB.GetUserByID(string(session.UserID))
				if err != nil {
					log.Printf("error getting user: %s", err.Error())
					logoutHandler(w, r)
					return
				}

				if !strings.HasPrefix(r.URL.Path, "/register/") && !strings.HasPrefix(r.URL.Path, "/login/") {
					log.Println(r.URL.Path)
					if len(user.Credentials) == 0 &&
						!strings.HasPrefix(r.URL.Path, "/register/") {
						// user did not complete registration
						log.Printf("user did not complete registration %s, deleting...", user.ID)
						err := userDB.DeleteUserByID(user.ID)
						if err != nil {
							log.Printf("error deleting user: %s", err.Error())
						}
						logoutHandler(w, r)
						return
					}
					if !authd && !strings.HasPrefix(r.URL.Path, "/login/") {
						// user did not complete login
						log.Printf("user not authenticated, logging out")
						logoutHandler(w, r)
						return
					}
				}

				r = r.WithContext(context.WithValue(r.Context(), "userID", user.ID))
				r = r.WithContext(context.WithValue(r.Context(), "userName", user.Username))
			}

			next.ServeHTTP(w, r)
		})
	}
}
