package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSSHandler(t *testing.T) {
	etag, handler, err := BuildCSSHandler(&Options{})
	if err != nil {
		t.Fatalf("failed to build css handler: %s", err)
	}

	if etag == "" {
		t.Fatalf("expected etag to be set")
	}

	req, err := http.NewRequest("GET", "/styles.css?etag=test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	req.Header.Set("If-None-Match", etag)

	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Fatalf("expected status code to be 304, got %d", rr.Code)
	}

	req, err = http.NewRequest("GET", "/styles.css", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rr = httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code to be 200, got %d", rr.Code)
	}

	if rr.Header().Get("ETag") != etag {
		t.Fatalf("expected etag to be set")
	}

	if !strings.Contains(rr.Body.String(), "TACHYONS") {
		t.Fatalf("expected body to contain TACHYONS")
	}
}

func TestJSHandler(t *testing.T) {
	etag, handler, err := BuildJSHandler(&Options{})
	if err != nil {
		t.Fatalf("failed to build js handler: %s", err)
	}

	if etag == "" {
		t.Fatalf("expected etag to be set")
	}

	req, err := http.NewRequest("GET", "/scripts.js?etag=test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	req.Header.Set("If-None-Match", etag)

	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Fatalf("expected status code to be 304, got %d", rr.Code)
	}

	req, err = http.NewRequest("GET", "/scripts.js", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rr = httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code to be 200, got %d", rr.Code)
	}

	if rr.Header().Get("ETag") != etag {
		t.Fatalf("expected etag to be set")
	}

	if !strings.Contains(rr.Body.String(), "WebAuthn") {
		t.Fatalf("expected body to contain console.log")
	}
}
