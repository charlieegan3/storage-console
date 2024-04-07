package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/charlieegan3/storage-console/pkg/config"
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

	mux := http.NewServeMux()

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
