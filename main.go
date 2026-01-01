package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {

	const filepathRoot = "."
	const port = "8080"
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
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
	mux.HandleFunc("POST /api/validate_chirp", handleValidateChirp)

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

func handleValidateChirp(w http.ResponseWriter, req *http.Request) {

	params, err := extractParams(validateChirpParameters{}, req)
	if err != nil {

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return
	}

	if len(params.Body) > VALID_CHIRP_LENGTH {
		log.Printf("Invalid Chirp")

		sendJSONResponse(
			w,
			http.StatusBadRequest,
			jsonErr{Error: "Chirp is too long"},
		)

		return
	}

	sendJSONResponse(
		w,
		http.StatusOK,
		validateChirpResponse{CleanedBody: cleanChirp(params.Body)},
	)

}
