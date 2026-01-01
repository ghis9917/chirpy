package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func sendResponse(w http.ResponseWriter, contentType string, statusCode int, content []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	w.Write(content)
}

func extractParams[T any](params T, req *http.Request) (T, error) {

	decoder := json.NewDecoder(req.Body)

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		return params, fmt.Errorf("Error decoding parameters: %s", err)
	}

	return params, nil
}

func sendJSONResponse[T any](w http.ResponseWriter, statusCode int, v T) {

	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendResponse(
		w,
		CONTENT_TYPE_JSON,
		statusCode,
		data,
	)

}

func cleanChirp(chirp string) string {

	words := strings.Split(chirp, " ")
	for i := range words {
		if PROFANE_WORDS[strings.ToLower(words[i])] {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}
