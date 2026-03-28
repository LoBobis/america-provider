package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// Server is the America webhook HTTP server.
// It is completely decoupled from any specific resource type.
//
// When America sends a deployment status callback to:
//
//	POST /webhook/{namespace}/{resourceName}
//
// the server pushes a GenericEvent into the shared event channel.
// Any controller that watches this channel via source.Channel + WatchesRawSource
// will receive the event and reconcile the matching resource.
type Server struct {
	logger  logging.Logger
	port    int
	server  *http.Server
	eventCh chan event.GenericEvent
}

// NewServer creates a new webhook server.
// eventCh is a shared channel — the same channel must be passed to each
// controller via source.Channel so they pick up the reconcile triggers.
func NewServer(logger logging.Logger, port int, eventCh chan event.GenericEvent) *Server {
	return &Server{
		logger:  logger,
		port:    port,
		eventCh: eventCh,
	}
}

// Start starts the webhook server. It blocks until the context is cancelled.
// Implements controller-runtime's Runnable interface.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/", s.handleWebhook)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.logger.Info("Starting America webhook server", "port", s.port)

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down webhook server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx) //nolint:errcheck
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleWebhook processes incoming webhooks from America.
// Expected URL format: POST /webhook/{namespace}/{resourceName}
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /webhook/{namespace}/{resourceName}
	path := strings.TrimPrefix(r.URL.Path, "/webhook/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		s.logger.Info("Invalid webhook path", "path", r.URL.Path)
		http.Error(w, "expected path: /webhook/{namespace}/{name}", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	name := parts[1]

	s.logger.Info("Received America webhook",
		"namespace", namespace,
		"name", name,
	)

	// Build a GenericEvent with a minimal ObjectMeta carrying the namespace/name.
	// The controller watching this channel will use the namespace/name to
	// enqueue a reconcile.Request for the matching CR.
	obj := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	evt := event.GenericEvent{Object: obj}

	select {
	case s.eventCh <- evt:
		s.logger.Info("Pushed reconcile event",
			"namespace", namespace,
			"name", name,
		)
	default:
		s.logger.Info("Event channel full, dropping webhook event",
			"namespace", namespace,
			"name", name,
		)
		http.Error(w, "server busy", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}
