package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nft-proxy/internal/alchemy"
)

type Server struct {
	alchemyClient *alchemy.Client
	logger        *zap.Logger
	server        *http.Server
}

func NewServer(alchemyClient *alchemy.Client, logger *zap.Logger) *Server {
	return &Server{
		alchemyClient: alchemyClient,
		logger:        logger,
	}
}

func (s *Server) SetupRoutes(router *mux.Router) {
	router.PathPrefix("/{chain}/{network}/nft/v3/").HandlerFunc(s.handleProxy)
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain := vars["chain"]
	network := vars["network"]

	alchemyPath := ExtractAlchemyPath(r)
	if alchemyPath == "" {
		s.logger.Error("Failed to extract Alchemy path from request")
		s.writeError(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	var respBody []byte
	var statusCode int
	var err error

	switch r.Method {
	case http.MethodGet:
		respBody, statusCode, err = s.alchemyClient.ProxyGET(
			r.Context(),
			chain,
			network,
			alchemyPath,
			r.URL.RawQuery,
		)
	case http.MethodPost:
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			s.logger.Error("Failed to read request body", zap.Error(readErr))
			s.writeError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		respBody, statusCode, err = s.alchemyClient.ProxyPOST(
			r.Context(),
			chain,
			network,
			alchemyPath,
			body,
		)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		s.logger.Error("Alchemy API error", zap.Error(err))
		s.writeError(w, "Failed to proxy request", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBody)
}

func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StartUnixSocket starts the HTTP server on a Unix socket
func (s *Server) StartUnixSocket(socketPath string) error {
	if err := os.RemoveAll(socketPath); err != nil {
		s.logger.Warn("Failed to remove existing socket file", zap.String("path", socketPath), zap.Error(err))
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Set socket permissions (readable/writable by all users for cross-container access)
	if err := os.Chmod(socketPath, 0666); err != nil {
		s.logger.Warn("Failed to set socket permissions", zap.String("path", socketPath), zap.Error(err))
	}

	router := mux.NewRouter()
	s.SetupRoutes(router)

	s.server = &http.Server{
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting NFT proxy server on Unix socket", zap.String("socket_path", socketPath))
	return s.server.Serve(listener)
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping NFT proxy HTTP server")

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}
