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
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
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

	mux.HandleFunc("GET /admin/metrics", apiCfg.middlewareMetricsGet)
	mux.HandleFunc("POST /admin/reset", apiCfg.middlewareMetricsReset)

	mux.HandleFunc("GET /api/healthz", handleReadiness)
	mux.HandleFunc("GET /api/chirps", apiCfg.middlewareGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.middlewareGetChirpByID)
	mux.HandleFunc("POST /api/chirps", apiCfg.middlewareCreateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.middlewareCreateUser)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())

}

func handleReadiness(w http.ResponseWriter, req *http.Request) {
	sendResponse(
		w,
		CONTENT_TYPE_PLAIN_TEXT,
		http.StatusOK,
		[]byte(http.StatusText(http.StatusOK)),
	)
}
