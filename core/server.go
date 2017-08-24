package core

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/go-redis/redis"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/starptech/go-web/cache"
	"github.com/starptech/go-web/config"
	"github.com/starptech/go-web/models"
)

type Server struct {
	Echo   *echo.Echo            // HTTP middleware
	config *config.Configuration // Configuration
	db     *models.Model         // Database connection
	cache  *redis.Client         // Redis cache connection
}

// NewServer will create a new instance of the application
func NewServer(config *config.Configuration) *Server {
	server := &Server{}
	server.config = config
	server.Echo = NewRouter(server)
	server.db = models.NewModel()
	server.cache = cache.NewCache(config)
	err := server.db.OpenWithConfig(config)

	if err != nil {
		log.Errorf("gorm: could not connect to db %q", err)
	}

	return server
}

// GetDB returns gorm (ORM)
func (s *Server) GetDB() *models.Model {
	return s.db
}

// GetCache returns the current redis client
func (s *Server) GetCache() *redis.Client {
	return s.cache
}

func (s *Server) GetConfig() *config.Configuration {
	return s.config
}

// Start the http server
func (s *Server) Start(addr string) error {
	return s.Echo.Start(addr)
}

// ServeStaticFiles serve static files for development purpose
func (s *Server) ServeStaticFiles() {
	s.Echo.Static("/", s.config.AssetsBuildDir)
}

// GracefulShutdown Wait for interrupt signal
// to gracefully shutdown the server with a timeout of 5 seconds.
func (s *Server) GracefulShutdown() {
	quit := make(chan os.Signal)

	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// close cache
	if s.cache != nil {
		cErr := s.cache.Close()
		if cErr != nil {
			s.Echo.Logger.Fatal(cErr)
		}
	}

	// close database connection
	if s.db != nil {
		dErr := s.db.Close()
		if dErr != nil {
			s.Echo.Logger.Fatal(dErr)
		}
	}

	// shutdown http server
	if err := s.Echo.Shutdown(ctx); err != nil {
		s.Echo.Logger.Fatal(err)
	}
}