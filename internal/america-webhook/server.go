package americawebhook

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type Server struct {
	// Channel that feeds GenericEvents into the reconciler
	Events chan event.GenericEvent
	Client client.Client
}

type IncomingEvent struct {
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`
	ExternalID   string `json:"externalId"`
	Status       string `json:"status"` // "created", "deleted", "updated", "failed"
}

func (s *Server) Start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleEvent)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("Webhook server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Webhook server failed: %v", err)
	}
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 1. Decode the payload from your cloud provider
	var payload IncomingEvent
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook: resource=%s status=%s", payload.ResourceName, payload.Status)

	// 2. Build a minimal object reference so controller-runtime
	//    knows WHICH managed resource to reconcile
	obj := &unstructured.Unstructured{}
	obj.SetName(payload.ResourceName)
	obj.SetNamespace(payload.Namespace)

	// 3. Push into the channel → wakes up the reconciler immediately
	s.Events <- event.GenericEvent{Object: obj}

	w.WriteHeader(http.StatusOK)
}
