package public

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/foolin/goview"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"

	"github.com/charlieegan3/curry-club/pkg/tool/stores"
	"github.com/charlieegan3/curry-club/pkg/tool/types"
	"github.com/charlieegan3/curry-club/pkg/tool/views"
)

func BuildProfileHandler(userDB *stores.UserDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("userID").(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		user, err := userDB.GetUserByID(userID)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		err = views.Engine.Render(
			w,
			http.StatusOK,
			"public/profile",
			goview.M{
				"user": user,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func BuildRegisterUserHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value("userID").(string)
		if ok {
			http.Redirect(w, r, "/profile", http.StatusFound)
			return
		}

		err := views.Engine.Render(
			w,
			http.StatusOK,
			"public/register",
			goview.M{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func BuildLoginUserHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value("userID").(string)
		if ok {
			http.Redirect(w, r, "/profile", http.StatusFound)
			return
		}

		err := views.Engine.Render(
			w,
			http.StatusOK,
			"public/login",
			goview.M{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func BuildLogoutUserHandler(sessionDB *stores.SessionDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// this shouldn't happen since the link is also annotated correctly
		if r.Header.Get("HX-Preload") != "" {
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
	}
}

func BuildRegisterUserBeginHandler(web *webauthn.WebAuthn, sessionDB *stores.SessionDB, userDB *stores.UserDB, db *sql.DB) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func BuildRegisterUserFinishHandler(web *webauthn.WebAuthn, sessionDB *stores.SessionDB, userDB *stores.UserDB, db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func BuildLoginUserBeginHandler(web *webauthn.WebAuthn, sessionDB *stores.SessionDB, userDB *stores.UserDB, db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func BuildLoginUserFinishHandler(web *webauthn.WebAuthn, sessionDB *stores.SessionDB, userDB *stores.UserDB, db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

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
	}
}

func jsonResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.Marshal(d)
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}
