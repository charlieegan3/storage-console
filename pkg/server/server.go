package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

func NewServer(cfg *config.Config) (Server, error) {
	return Server{cfg: cfg}, nil
}

type Server struct {
	cfg        *config.Config
	httpServer *http.Server
}

func (s *Server) Start(ctx context.Context) error {
	var err error

	mux, err := newMux(
		&handlers.Options{
			DevMode: s.cfg.Server.DevMode,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create mux: %w", err)
	}

	s.httpServer = &http.Server{
		Addr: fmt.Sprintf(
			"%s:%d",
			s.cfg.Server.Address,
			s.cfg.Server.Port,
		),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		err = s.httpServer.Shutdown(ctx)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		err = s.httpServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	s.httpServer = nil

	return nil
}
