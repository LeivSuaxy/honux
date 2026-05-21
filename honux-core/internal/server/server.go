package server

import (
	"context"
	"database/sql"
	"fmt"
	"honux-core/internal/db/repository"
	http_users "honux-core/internal/http-api/modules/users/handlers"
	"honux-core/internal/service"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(port int, db *sql.DB) *Server {
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)
	userHandler := http_users.NewUserHandlerHTTP(userSvc)

	mux := http.NewServeMux()
	registerRoutes(mux, userHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

}

func registerRoutes(mux *http.ServeMux, u *http_users.UserHandlerHTTP) {
	mux.HandleFunc("GET /health", healthCheck)
	mux.HandleFunc("POST /api/v1/users", u.Create)
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
