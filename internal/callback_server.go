package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type CallbackServer struct {
	port     int
	samlChan chan string
	errChan  chan error
	server   *http.Server
	config   *Config
	gotSAML  bool
}

func NewCallbackServer(cfg *Config) *CallbackServer {
	return &CallbackServer{
		samlChan: make(chan string, 1),
		errChan:  make(chan error, 1),
		config:   cfg,
	}
}
func (s *CallbackServer) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	if s.port == 0 {
		addr = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.port = listener.Addr().(*net.TCPAddr).Port
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	mux.HandleFunc("/status", s.handleStatus)
	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.errChan <- err
		}
	}()
	return nil
}
func (s *CallbackServer) GetPort() int {
	return s.port
}
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		http.Error(w, "Missing SAML", http.StatusBadRequest)
		return
	}
	select {
	case s.samlChan <- samlResponse:
		s.gotSAML = true
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html><html><body><h1>✓ Success</h1><p>You can close this window.</p></body></html>`)
	default:
		http.Error(w, "Busy", http.StatusServiceUnavailable)
	}
}
func (s *CallbackServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.gotSAML {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
func (s *CallbackServer) WaitForSAML(timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case saml := <-s.samlChan:
		return saml, nil
	case err := <-s.errChan:
		return "", err
	case <-timer.C:
		return "", fmt.Errorf("timeout")
	}
}
func (s *CallbackServer) Shutdown() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
