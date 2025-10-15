package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/starwalkn/kairyu"
)

func StartServer(cfg *kairyu.GatewayConfig) {
	if !cfg.AdminPanel.Enable {
		fmt.Printf("📊 Admin dashboard disabled\n")
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	})

	staticDir := filepath.Join("/", "app", "admin", "static")
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fs)

	addr := fmt.Sprintf(":%d", cfg.AdminPanel.Port)
	fmt.Printf("📊 Admin dashboard available at http://localhost%s\n", addr)

	go http.ListenAndServe(addr, mux)
}
