package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ghis9917/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {

	godotenv.Load()
	const filepathRoot = "."
	const port = "8080"

	platform := os.Getenv("PLATFORM")
	dbURL := os.Getenv("DB_URL")
	serverSecret := os.Getenv("SERVER_SECRET")
	polkaSecret := os.Getenv("POLKA_KEY")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
		serverSecret:   serverSecret,
		polkaSecret:    polkaSecret,
	}

	mux := http.NewServeMux()
	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(
			http.StripPrefix(
				"/app",
				http.FileServer(
					http.Dir(filepathRoot),
				),
			),
		),
	)
	// ============ ADMIN =============
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handleResetMetrics)
	// ============ API GET =============
	mux.HandleFunc("GET /api/healthz", handleReadiness)
	mux.HandleFunc("GET /api/chirps", apiCfg.handleGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handleGetChirpByID)
	// ============ API POST =============
	mux.HandleFunc("POST /api/chirps", apiCfg.handleCreateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.handleCreateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handleLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handleRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handleRevoke)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handleUpgradeUser)
	// ============ API PUT =============
	mux.HandleFunc("PUT /api/users", apiCfg.handleUpdateUser)
	// ============ API DELETE =============
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handleDeleteChirpByID)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())

}
