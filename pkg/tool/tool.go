package tool

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/go-webauthn/webauthn/protocol"
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
	"github.com/charlieegan3/curry-club/pkg/tool/types"
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
	web, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Curry Club",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
	})
	if err != nil {
		return fmt.Errorf("error creating webauthn: %w", err)
	}

	path := "web.auth.username"
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

	router.HandleFunc("/register/begin/{username}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		log.Println("beginning registration for:", username)

		user, err := userDB.GetUser(username)
		if err != nil {
			log.Println("creating new user: ", username)
			user = types.NewUser(username)
			err := userDB.PutUser(user)
			if err != nil {
				log.Println(err)
				jsonResponse(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// validate that the user is logged in before adding more credentials
			cookie, err := r.Cookie("session")
			if err == nil {
				// validate the session
				session, err := sessionDB.GetSession(cookie.Value)
				if err != nil {
					fmt.Println(err)
					http.Redirect(w, r, "/logout", http.StatusFound)
					return
				}

				sessionUser, err := userDB.GetUserByID(string(session.UserID))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				if user.ID != sessionUser.ID {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("forbidden"))
					return
				}
			} else {
				log.Println("session missing, can't add more credentials to user")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("forbidden"))
				return
			}
		}

		registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
			credCreationOpts.CredentialExcludeList = user.CredentialExcludeList()
		}

		options, sessionData, err := web.BeginRegistration(
			user,
			registerOptions,
		)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sessionID, err := sessionDB.StartSession(sessionData)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionID,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})

		jsonResponse(w, options, http.StatusOK)
	})

	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		// this shouldn't happen since the link is also annotated correctly
		if r.Header.Get("HX-Preload") != "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		cookie, err := r.Cookie("session")
		if err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:    "session",
				Value:   "",
				Expires: time.Unix(0, 0),
			})
			err = sessionDB.DeleteSession(cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	router.HandleFunc("/register/finish/{username}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		log.Println("finalising registration for: ", username)

		user, err := userDB.GetUser(username)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			log.Println("cookie:", err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		sessionData, err := sessionDB.GetSession(cookie.Value)
		if err != nil {
			log.Println("cookie:", err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		credential, err := web.FinishRegistration(user, *sessionData, r)
		if err != nil {
			log.Println("finalising: ", err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = userDB.AddCredentialsForUser(user, []webauthn.Credential{*credential})
		if err != nil {
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, "Registration Success", http.StatusOK)
	})

	router.HandleFunc("/login/begin/{username}", func(w http.ResponseWriter, r *http.Request) {
		// get username
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		log.Println("user: ", username, "logging in")

		// get user
		user, err := userDB.GetUser(username)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		// generate PublicKeyCredentialRequestOptions, session data
		options, sessionData, err := web.BeginLogin(user)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sessionID, err := sessionDB.StartSession(sessionData)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionID,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})

		jsonResponse(w, options, http.StatusOK)
	})

	router.HandleFunc("/login/finish/{username}", func(w http.ResponseWriter, r *http.Request) {

		// get username
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		log.Println("user: ", username, "finishing logging in")
		// get user
		user, err := userDB.GetUser(username)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		// load the session data
		cookie, err := r.Cookie("session")
		if err != nil {
			log.Println("cookie:", err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		sessionData, err := sessionDB.GetSession(cookie.Value)
		if err != nil {
			log.Println("session:", err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		c, err := web.FinishLogin(user, *sessionData, r)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		if c.Authenticator.CloneWarning {
			log.Println("cloned key detected")
			jsonResponse(w, "cloned key detected", http.StatusBadRequest)
			return
		}

		jsonResponse(w, "Login Success", http.StatusOK)
	})

	router.HandleFunc("/", public.BuildIndexHandler(sessionDB, userDB, w.db)).Methods("GET")

	router.Use(middlewares.BuildGoAwayMiddleware())
	router.Use(gorillaHandlers.CompressHandler)
	router.NotFoundHandler = http.HandlerFunc(status.BuildNotFoundHandler(w.db))

	return nil
}
func (w *Website) HTTPHost() string {
	path := "web.host"
	host, ok := w.config.Path(path).Data().(string)
	if !ok {
		return "example.com"
	}
	return host
}
func (w *Website) HTTPPath() string { return "" }

func (w *Website) ExternalJobsFuncSet(f func(job apis.ExternalJob) error) {}

func jsonResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.Marshal(d)
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}
