package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type deployment struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Parameters map[string]interface{} `json:"parameters"`
}

var (
	deployments = map[string]*deployment{}
	mu          sync.Mutex
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			handleCreate(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/deployments/") {
			handleGet(w, r)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/operations" {
			handleCreateOperation(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/operations/") {
			handleGetOperation(w, r)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Mock America API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resource_type"`
		Name         string `json:"name"`
		Parameters   string `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	mu.Lock()
	deployments[id] = &deployment{
		ID:     id,
		Name:   req.Name,
		Status: "CREATED",
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"deployment_id": id,
		"message":       "created",
	})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/deployments/")

	mu.Lock()
	dep, ok := deployments[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dep)
}

func handleCreateOperation(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"operation_id": id,
		"status":       "COMPLETED",
	})
}

func handleGetOperation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/operations/")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operation_id": id,
		"status":       "COMPLETED",
		"result":       map[string]string{"message": "done"},
	})
}

func init() {
	fmt.Println("Mock America API server starting...")
}
