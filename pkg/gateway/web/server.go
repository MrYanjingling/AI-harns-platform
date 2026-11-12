package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"harns-platform/cmd/gateway/config"
	"harns-platform/cmd/gateway/options"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/generic"
	"harns-platform/pkg/logging"
)

type Server struct {
	*generic.Server
	*config.Config
}

func NewServer(router *gin.Engine, o *options.Options, config *config.Config) (*Server, error) {
	router.Use(generic.CORS(config.CORS))
	router.Use(gin.Recovery())
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		response.WriteError(c, http.StatusMethodNotAllowed, response.ErrMethodNotAllowed())
	})

	s := &generic.Server{
		Router: router,
		Port:   o.Port,
	}

	server := &Server{
		Server: s,
		Config: config,
	}
	router.NoRoute(server.handleNoRoute)
	if config.UIDir != "" {
		if _, err := os.Stat(filepath.Join(config.UIDir, "index.html")); err != nil {
			logging.Warnf("Gateway UI assets not found at %q: %v", config.UIDir, err)
		}
	}

	server.InstallHandlers()

	return server, nil

}

func (s *Server) handleNoRoute(c *gin.Context) {
	if s.serveUI(c) {
		return
	}
	response.WriteError(c, http.StatusNotFound, response.ErrNotFound())
}

func (s *Server) serveUI(c *gin.Context) bool {
	if s == nil || strings.TrimSpace(s.Config.UIDir) == "" {
		return false
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api/") || path == "/api" {
		return false
	}

	dir, err := filepath.Abs(s.Config.UIDir)
	if err != nil {
		return false
	}
	indexPath := filepath.Join(dir, "index.html")
	if info, err := os.Stat(indexPath); err != nil || info.IsDir() {
		return false
	}

	clean := strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	if clean == "" || clean == "." {
		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
		return true
	}
	target := filepath.Join(dir, filepath.FromSlash(clean))
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		if strings.HasPrefix(clean, "assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.File(target)
		return true
	}
	if strings.HasPrefix(clean, "assets/") {
		return false
	}

	c.Header("Cache-Control", "no-cache")
	c.File(indexPath)
	return true
}

func (s *Server) InstallHandlers() {
	v1 := s.Router.Group("/api/v1")
	v1.Use(generic.BodyLimit(s.Config.MaxBodyBytes))
	// Legacy device manager endpoints are disabled in favor of collections.
	// device.InstallHandler(v1, s.Config.DeviceMgr)
	gateway.InstallHandler(v1, s.Config.GatewayMgr)
	drivermgr.InstallHandler(v1, s.Config.DriverMgr)
	collectionmgr.InstallHandler(v1, s.Config.CollectionMgr)
	pushmgr.InstallHandler(v1, s.Config.PushMgr)
}

func (s *Server) Serve() (func(ctx context.Context), error) {
	var srv *http.Server
	if len(s.Config.CertFile) != 0 && len(s.Config.KeyFile) != 0 {
		x509KeyPair, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return nil, err
		}
		c := &tls.Config{
			Certificates: []tls.Certificate{x509KeyPair},
		}

		srv = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.Port),
			Handler:           s.Router,
			TLSConfig:         c,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			logging.Error(srv.ListenAndServeTLS("", ""))
		}()
	} else {
		srv = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.Port),
			Handler:           s.Router,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			logging.Error(srv.ListenAndServe())
		}()
	}

	return func(ctx context.Context) {
		srv.SetKeepAlivesEnabled(false)
		// Drain HTTP requests before tearing down the legacy device manager.
		if err := srv.Shutdown(ctx); err != nil {
			logging.Error(err)
		}
		if s.Config.DeviceMgr != nil {
			if err := s.Config.DeviceMgr.Shutdown(ctx); err != nil {
				logging.Error(err)
			}
		}
		if s.Config.PushMgr != nil {
			s.Config.PushMgr.Close(ctx)
		}
		if s.Config.DB != nil {
			if err := s.Config.DB.Close(); err != nil {
				logging.Error(err)
			}
		}
	}, nil
}
