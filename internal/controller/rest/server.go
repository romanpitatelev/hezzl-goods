package rest

import (
	"context"
	"crypto/rsa"
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
	cfg    Config
	server *http.Server
	// usersHandler usersHandler
	// taskHandler  taskHandler
	key *rsa.PublicKey
}

// type usersHandler interface {
// 	GetUser(w http.ResponseWriter, r *http.Request)
// 	UpdateUser(w http.ResponseWriter, r *http.Request)
// 	DeleteUser(w http.ResponseWriter, r *http.Request)
// 	GetUsers(w http.ResponseWriter, r *http.Request)
// 	GetTopUsers(w http.ResponseWriter, r *http.Request)
// }

// type taskHandler interface {
// 	Task(w http.ResponseWriter, r *http.Request)
// 	ReferralTask(w http.ResponseWriter, r *http.Request)
// }

func New(
	cfg Config,
	// usersHandler usersHandler,
	// taskHandler taskHandler,
	key *rsa.PublicKey,
) *Server {
	router := chi.NewRouter()
	s := &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.BindAddress,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeoutValue,
		},
		// usersHandler: usersHandler,
		// taskHandler:  taskHandler,
		key: key,
	}

	router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Use(middleware.Recoverer)
			r.Use(s.jwtAuth)

			// r.Get("/users/{id}/status", s.usersHandler.GetUser)
			// r.Patch("/users/{id}", s.usersHandler.UpdateUser)
			// r.Delete("/users/{id}", s.usersHandler.DeleteUser)
			// r.Get("/users", s.usersHandler.GetUsers)
			// r.Get("/users/leaderboard", s.usersHandler.GetTopUsers)
			// r.Post("/users/{id}/task/complete", s.taskHandler.Task)
			// r.Post("/users/{id}/referrer", s.taskHandler.ReferralTask)
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
