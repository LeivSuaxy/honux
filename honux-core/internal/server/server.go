package server

import (
	"context"
	"database/sql"
	"fmt"
	"honux-core/internal/db/repository"
	"honux-core/internal/http-api/middlewares"
	http_floors "honux-core/internal/http-api/modules/floors"
	http_users "honux-core/internal/http-api/modules/users"
	"honux-core/internal/service"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(port int, db *sql.DB) *Server {
	// Initialize Dependencies
	// User
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)
	userHandler := http_users.NewUserHandlerHTTP(userSvc)

	// Floor
	floorRepo := repository.NewFloorRepository(db)
	floorSvc := service.NewFloorService(floorRepo)
	floorHandler := http_floors.NewFloorHandlerHTTP(floorSvc)

	// Create MUX Server
	mux := http.NewServeMux()

	stack := middlewares.Chain(
		middlewares.Recover,
		middlewares.RequestID,
		middlewares.Logger,
	)

	// All Routes
	http_users.RegisterRoutes(mux, userHandler)
	http_floors.RegisterRoutes(mux, floorHandler)
	registerRoutes(mux)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      stack(mux),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthCheck)
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
