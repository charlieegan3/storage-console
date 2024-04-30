package utils

import (
	"net/http"
)

func SetCacheControl(w http.ResponseWriter, cacheControl string) {
	w.Header().Set("Cache-Control", cacheControl)
}
