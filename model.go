package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})

}

func (cfg *apiConfig) middlewareMetricsGet(w http.ResponseWriter, req *http.Request) {

	sendResponse(
		w,
		CONTENT_TYPE_HTML,
		http.StatusOK,
		[]byte(fmt.Sprintf(METRICS_HTML, cfg.fileserverHits.Load())),
	)

}

func (cfg *apiConfig) middlewareMetricsReset(w http.ResponseWriter, req *http.Request) {

	cfg.fileserverHits.Store(0)

	sendResponse(
		w,
		CONTENT_TYPE_PLAIN_TEXT,
		http.StatusOK,
		[]byte(fmt.Sprintf("Hits: %v\n", cfg.fileserverHits.Load())),
	)

}

type validateChirpParameters struct {
	Body string `json:"body"`
}

type validateChirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

type jsonErr struct {
	Error string `json:"error"`
}
