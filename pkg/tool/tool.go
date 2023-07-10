package tool

import (
	"database/sql"
	"embed"
	"fmt"
	"net/http"

	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/go-webauthn/webauthn/webauthn"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"

	"github.com/charlieegan3/curry-club/pkg/tool/handlers"
	"github.com/charlieegan3/curry-club/pkg/tool/handlers/admin"
	"github.com/charlieegan3/curry-club/pkg/tool/handlers/public"
	"github.com/charlieegan3/curry-club/pkg/tool/handlers/status"
	"github.com/charlieegan3/curry-club/pkg/tool/middlewares"
	"github.com/charlieegan3/curry-club/pkg/tool/stores"
)

//go:embed migrations
var migrations embed.FS

var (
	web *webauthn.WebAuthn
	err error
)

// Website is a tool that runs curry-club.org
type Website struct {
	db     *sql.DB
	config *gabs.Container

	bucketName string
	googleJSON string
	adminPath  string
}

func (w *Website) Name() string {
	return "curry-club"
}

func (w *Website) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		HTTP:     true,
		HTTPHost: true,
		Config:   true,
		Database: true,
	}
}

func (w *Website) DatabaseMigrations() (*embed.FS, string, error) {
	return &migrations, "migrations", nil
}

func (w *Website) DatabaseSet(db *sql.DB) {
	w.db = db
}

func (w *Website) SetConfig(config map[string]any) error {
	var ok bool
	var path string
	w.config = gabs.Wrap(config)

	path = "web.admin_path"
	w.adminPath, ok = w.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("config value %s not set", path)
	}

	return nil
}

func (w *Website) Jobs() ([]apis.Job, error) { return []apis.Job{}, nil }

func (w *Website) HTTPAttach(router *mux.Router) error {
	var path string

	path = "webauthn.host"
	webAuthnHost, ok := w.config.Path(path).Data().(string)
	if !ok {
		webAuthnHost = "localhost"
	}
	path = "webauthn.origin"
	webAuthnOrigin, ok := w.config.Path(path).Data().(string)
	if !ok {
		webAuthnHost = "http://localhost:3000"
	}

	web, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Curry Club",
		RPID:          webAuthnHost,
		RPOrigins:     []string{webAuthnOrigin},
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
	adminRouter.Use(middlewares.InitMiddlewareAuth(adminUsername, adminPassword))

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
	router.HandleFunc("/profile", public.BuildProfileHandler(userDB)).Methods("GET")

	router.HandleFunc("/", public.BuildIndexHandler(sessionDB, userDB, w.db)).Methods("GET")

	router.Use(middlewares.BuildGoAwayMiddleware())
	router.Use(gorillaHandlers.CompressHandler)
	router.Use(middlewares.BuildSessionMiddleware(sessionDB, userDB))
	router.NotFoundHandler = http.HandlerFunc(status.BuildNotFoundHandler(w.db))

	return nil
}
func (w *Website) HTTPHost() string {
	path := "web.host"
	host, ok := w.config.Path(path).Data().(string)
	if !ok {
		return "localhost"
	}
	return host
}
func (w *Website) HTTPPath() string { return "" }

func (w *Website) ExternalJobsFuncSet(f func(job apis.ExternalJob) error) {}
