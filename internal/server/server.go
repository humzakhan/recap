package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/humzakhan/recap/internal/aiclient"
	"github.com/humzakhan/recap/internal/config"
)

// Server holds the Gin router and application configuration.
type Server struct {
	Router   *gin.Engine
	Config   *config.Config
	AIClient aiclient.AIClient
}

// New creates a new Server with the given configuration, sets up CORS and routes.
func New(cfg *config.Config, client aiclient.AIClient) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	s := &Server{Router: router, Config: cfg, AIClient: client}
	s.setupCORS()
	s.setupRoutes()

	return s
}

// Run starts the HTTP server on the configured host and port.
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.Config.Server.Host, s.Config.Server.Port)
	return s.Router.Run(addr)
}

// setupCORS configures CORS middleware. If the RECAP_ALLOWED_ORIGINS env var is
// set (comma-separated), those origins are used. Otherwise, localhost:3000 and
// 127.0.0.1:3000 are allowed by default.
func (s *Server) setupCORS() {
	var allowedOrigins []string

	if envOrigins := os.Getenv("RECAP_ALLOWED_ORIGINS"); envOrigins != "" {
		for _, o := range strings.Split(envOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	} else {
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	s.Router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
}

// setupRoutes registers all API route handlers.
func (s *Server) setupRoutes() {
	rateLimit := parseRateLimit()

	v1 := s.Router.Group("/api/v1")
	v1.GET("/health", s.handleHealth)
	v1.POST("/extract", RateLimiter(rateLimit), s.handleExtract)
	v1.GET("/templates", s.handleListTemplates)
	v1.GET("/templates/:name", s.handleGetTemplate)
	v1.POST("/templates", s.handleCreateTemplate)
	v1.DELETE("/templates/:name", s.handleDeleteTemplate)
	v1.GET("/models", s.handleListModels)
}

// parseRateLimit reads the RECAP_RATE_LIMIT env var, defaulting to 10 requests
// per minute if unset or invalid.
func parseRateLimit() int {
	if v := os.Getenv("RECAP_RATE_LIMIT"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 10
}
