package mux

import "net/http"

func NewMux() *Mux {
	return &Mux{}
}

type Mux struct {
	mux http.ServeMux
}
