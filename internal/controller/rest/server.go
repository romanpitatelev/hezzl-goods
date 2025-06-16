package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const (
	readHeaderTimeoutValue = 3 * time.Second
	timeoutDuration        = 10 * time.Second
)

type Config struct {
	BindAddress string
}

type Server struct {
	cfg          Config
	server       *http.Server
	goodsHandler goodsHandler
}

type goodsHandler interface {
	CreateGood(w http.ResponseWriter, r *http.Request)
	GetGood(w http.ResponseWriter, r *http.Request)
	UpdateGood(w http.ResponseWriter, r *http.Request)
	DeleteGood(w http.ResponseWriter, r *http.Request)
	GetGoods(w http.ResponseWriter, r *http.Request)
	Reprioritize(w http.ResponseWriter, r *http.Request)
}

func New(
	cfg Config,
	goodsHandler goodsHandler,
) *Server {
	router := chi.NewRouter()
	s := &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.BindAddress,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeoutValue,
		},
		goodsHandler: goodsHandler,
	}

	router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Use(middleware.Recoverer)

			r.Post("/good/create", s.goodsHandler.CreateGood)
			r.Get("/good/get", s.goodsHandler.GetGood)
			r.Patch("/good/update", s.goodsHandler.UpdateGood)
			r.Delete("/good/remove", s.goodsHandler.DeleteGood)
			r.Get("/goods/list", s.goodsHandler.GetGoods)
			r.Patch("/good/reprioritize", s.goodsHandler.Reprioritize)
		})
	})

	return s
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		gracefulCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		//nolint:contextcheck
		if err := s.server.Shutdown(gracefulCtx); err != nil {
			log.Warn().Err(err).Msg("failed to shutdown server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}

	return nil
}
