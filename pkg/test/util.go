package test

import (
	"io"
	"testing"
)

// TLogWriter is an io.Writer that writes to a testing.T log.
type TLogWriter struct {
	t *testing.T
}

// Write writes the given data to the testing.T log.
func (w *TLogWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

// NewTLogWriter creates a new TLogWriter for the given testing.T.
func NewTLogWriter(t *testing.T) io.Writer {
	return &TLogWriter{t: t}
}
