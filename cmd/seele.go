package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zerothy/seele/model"
	"github.com/zerothy/seele/service"
)

const gracefulShutdownTimeout = 5 * time.Second

type Server struct {
	store *service.Store
	ring  *service.HashRing
}

func NewServer(store *service.Store, ring *service.HashRing) *Server {
	return &Server{store: store, ring: ring}
}

func (server *Server) HandleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}

	value, found := server.store.Get(key)
	if !found {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(model.GetResponse{Value: value}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (server *Server) HandleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := server.store.Set(req.Key, req.Value); err != nil {
		http.Error(w, "Failed to set value", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(model.SetResponse{Success: true}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (server *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}

	if err := server.store.Delete(key); err != nil {
		http.Error(w, "Failed to delete key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(model.DeleteResponse{Success: true}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (server *Server) HandleKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys := server.store.Keys()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (server *Server) HandleSyncMerkle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rootHash := server.store.GetMerkleRoot()
	response := map[string]string{"root_hash": rootHash}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func StartServer(port string, dataDir string, joinAddr string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	store, err := service.NewStore(dataDir)
	if err != nil {
		return err
	}

	ring := service.NewHashRing(50)

	var p2pPort int
	fmt.Sscanf(port, "%d", &p2pPort)
	p2pPort += 1000

	p2p, err := service.NewP2P(p2pPort, fmt.Sprintf("node-%s", port))
	if err != nil {
		return fmt.Errorf("Failed to start P2P: %v", err)
	}

	go func() {
		for {
			time.Sleep(1 * time.Second)
			members := p2p.Members()
			var activeShards []string
			for _, member := range members {
				parts := strings.Split(member, ":")
				if len(parts) != 2 || strings.Contains(member, "proxy") {
					continue
				}
				ip := parts[0]
				var gPort int
				fmt.Sscanf(parts[1], "%d", &gPort)
				httpPort := gPort - 1000
				activeShards = append(activeShards, fmt.Sprintf("%s:%d", ip, httpPort))
			}
			if len(activeShards) > 0 {
				ring.SetNodes(activeShards)
			}
		}
	}()

	fmt.Printf("P2P Service Started on port %d\n", p2pPort)

	if joinAddr != "" {
		go func() {
			time.Sleep(1 * time.Second)
			fmt.Printf("P2P: Attempting to join %s...\n", joinAddr)
			if err := p2p.Join([]string{joinAddr}); err != nil {
				fmt.Printf("P2P: Failed to join %s: %v\n", joinAddr, err)
			} else {
				fmt.Println("P2P: Successfully joined cluster!")
			}
		}()
	} else {
		fmt.Println("P2P: Starting as Seed Node")
	}

	server := NewServer(store, ring)

	mux := http.NewServeMux()
	mux.HandleFunc("/get", server.HandleGet)
	mux.HandleFunc("/set", server.HandleSet)
	mux.HandleFunc("/delete", server.HandleDelete)
	mux.HandleFunc("/keys", server.HandleKeys)
	mux.HandleFunc("/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(p2p.Members()); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/sync/merkle", server.HandleSyncMerkle)

	server.StartAntiEntropy(p2p, port)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		fmt.Println("Seele server listening on :" + port + "...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error starting server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	fmt.Println("\nShutting down gracefully...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancelShutdown()

	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		fmt.Println("Error shutting down server:", err)
	}

	p2p.Leave(1 * time.Second)
	p2p.Shutdown()

	fmt.Println("Server stopped gracefully.")
	return nil
}
