package main

import (
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/charlieegan3/storage-console/pkg/handlers/public"
	"github.com/charlieegan3/storage-console/pkg/middlewares"
	"github.com/charlieegan3/storage-console/pkg/stores"
)

func main() {
	var path string

	path = "webauthn.host"
	webAuthnHost, ok := w.config.Path(path).Data().(string)
	if !ok {
		webAuthnHost = "localhost"
	}
	path = "webauthn.origins"
	webAuthnOrigins, ok := w.config.Path(path).Data().([]string)
	if !ok {
		webAuthnOrigins = []string{"http://localhost:3000"}
	}

	web, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Curry Club",
		RPID:          webAuthnHost,
		RPOrigins:     webAuthnOrigins,
	})
	if err != nil {
		return fmt.Errorf("error creating webauthn: %w", err)
	}

	path = "web.auth.username"
	adminUsername, ok := w.config.Path(path).Data().(string)
	if !ok {
		adminUsername = "example"
	}

	path = "web.auth.password"
	adminPassword, ok := w.config.Path(path).Data().(string)
	if !ok {
		adminPassword = "example"
	}

	sessionDB := stores.NewSessionDB(w.db)
	userDB := stores.NewUsersDB(w.db)

	router.StrictSlash(true)
	adminRouter := router.PathPrefix(w.adminPath).Subrouter()
	adminRouter.StrictSlash(true) // since not inherited

	// admin routes -------------------------------------
	adminRouter.HandleFunc("/", admin.BuildIndexHandler(w.adminPath))
	adminRouter.HandleFunc("/blocks", admin.BuildBlockIndexHandler(w.db, w.adminPath)).Methods("GET")
	adminRouter.HandleFunc("/blocks", admin.BuildBlockCreateHandler(w.db, w.adminPath)).Methods("POST")
	adminRouter.HandleFunc("/blocks/{blockKey}", admin.BuildBlockUpdateHandler(w.db, w.adminPath)).Methods("POST")

	// public routes ------------------------------------
	router.HandleFunc("/favicon.ico", handlers.BuildFaviconHandler())
	router.HandleFunc("/robots.txt", handlers.BuildRobotsHandler())
	cssHandler, err := handlers.BuildCSSHandler()
	if err != nil {
		return err
	}
	router.HandleFunc("/styles.css", cssHandler).Methods("GET")

	jsHandler, err := handlers.BuildJSHandler()
	if err != nil {
		return err
	}
	router.HandleFunc("/script.js", jsHandler).Methods("GET")

	router.HandleFunc(
		"/fonts/{path:.*}",
		handlers.BuildFontHandler(),
	).Methods("GET")
	router.HandleFunc(
		"/static/{path:.*}",
		handlers.BuildStaticHandler(),
	).Methods("GET")

	router.HandleFunc("/register/begin/{username}", public.BuildRegisterUserBeginHandler(web, sessionDB, userDB, w.db)).Methods("GET")
	router.HandleFunc("/register/finish/{username}", public.BuildRegisterUserFinishHandler(web, sessionDB, userDB, w.db)).Methods("POST")
	router.HandleFunc("/login/begin/{username}", public.BuildLoginUserBeginHandler(web, sessionDB, userDB, w.db)).Methods("GET")
	router.HandleFunc("/login/finish/{username}", public.BuildLoginUserFinishHandler(web, sessionDB, userDB, w.db)).Methods("POST")
	router.HandleFunc("/login", public.BuildLoginUserHandler()).Methods("GET")
	router.HandleFunc("/logout", public.BuildLogoutUserHandler(sessionDB)).Methods("GET")
	router.HandleFunc("/register", public.BuildRegisterUserHandler()).Methods("GET")
	router.HandleFunc("/profile", public.BuildProfileHandler(userDB)).Methods("GET")

	router.HandleFunc("/", public.BuildIndexHandler(sessionDB, userDB, w.db)).Methods("GET")

	router.Use(middlewares.BuildGoAwayMiddleware())
	router.Use(gorillaHandlers.CompressHandler)
	router.Use(middlewares.BuildSessionMiddleware(sessionDB, userDB))
	router.NotFoundHandler = http.HandlerFunc(status.BuildNotFoundHandler(w.db))

	return nil
}
