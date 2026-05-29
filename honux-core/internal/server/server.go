package server

import (
	"context"
	"database/sql"
	"fmt"
	"honux-core/internal/db/repository"
	"honux-core/internal/http-api/middlewares"
	http_floors "honux-core/internal/http-api/modules/floors"
	http_users "honux-core/internal/http-api/modules/users"
	http_zones "honux-core/internal/http-api/modules/zones"
	"honux-core/internal/providers/cache"
	"honux-core/internal/server/router"
	"honux-core/internal/service"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(port int, db *sql.DB) *Server {
	start := time.Now()
	// Initialize Dependencies
	cacheProvider := cache.GetCache()
	cacheMiddleware := middlewares.NewCacheMiddleware(cacheProvider, 5*time.Minute)
	// User
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)
	userHandler := http_users.NewUserHandlerHTTP(userSvc)

	// Floor
	floorRepo := repository.NewFloorRepository(db)
	floorSvc := service.NewFloorService(floorRepo)
	floorHandler := http_floors.NewFloorHandlerHTTP(floorSvc)

	// Zone
	zoneRepo := repository.NewZoneRepository(db)
	zoneSvc := service.NewZoneService(zoneRepo)
	zoneHandler := http_zones.NewZoneHandlerHTTP(zoneSvc)

	// Create MUX Server
	mux := router.NewTrackedMux()

	stack := middlewares.Chain(
		middlewares.Recover,
		middlewares.RequestID,
		middlewares.Logger,
		cacheMiddleware,
	)

	// All Routes
	http_users.RegisterRoutes(mux, userHandler)
	http_floors.RegisterRoutes(mux, floorHandler)
	http_zones.RegisterRoutes(mux, zoneHandler)
	registerRoutes(mux)

	mux.PrintRoutes(time.Since(start).Milliseconds())

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

func registerRoutes(r router.Router) {
	m := r.Module("server")
	m.HandleFunc("GET /health", healthCheck)
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
