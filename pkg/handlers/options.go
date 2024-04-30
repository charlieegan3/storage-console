package handlers

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/charlieegan3/storage-console/pkg/stores"
)

type Options struct {
	DevMode    bool
	EtagScript string
	EtagStyles string

	WebAuthn     *webauthn.WebAuthn
	SessionStore stores.SessionStore
	UserStore    stores.UserStore
}
