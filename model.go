package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ghis9917/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
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

	if cfg.platform != "dev" {

		sendResponse(
			w,
			CONTENT_TYPE_PLAIN_TEXT,
			http.StatusForbidden,
			[]byte{},
		)

	}

	cfg.fileserverHits.Store(0)
	if err := cfg.db.DeleteAllUsers(req.Context()); err != nil {

		sendResponse(
			w,
			CONTENT_TYPE_PLAIN_TEXT,
			http.StatusInternalServerError,
			[]byte{},
		)

	}

	sendResponse(
		w,
		CONTENT_TYPE_PLAIN_TEXT,
		http.StatusOK,
		[]byte(fmt.Sprintf("Hits: %v\n", cfg.fileserverHits.Load())),
	)

}

func (cfg *apiConfig) middlewareCreateUser(w http.ResponseWriter, req *http.Request) {

	params, err := extractParams(createUserParameters{}, req)
	if err != nil {

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return
	}

	user, err := cfg.db.CreateUser(
		req.Context(),
		params.Email,
	)
	if err != nil {

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return

	}

	sendJSONResponse(
		w,
		http.StatusCreated,
		User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
	)

}

func (cfg *apiConfig) middlewareCreateChirp(w http.ResponseWriter, req *http.Request) {

	params, err := extractParams(createChirpParameters{}, req)
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

	chirp, err := cfg.db.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   cleanChirp(params.Body),
			UserID: params.UserID,
		},
	)
	if err != nil {

		log.Printf("Failed to save Chirp in DB")

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return

	}

	sendJSONResponse(
		w,
		http.StatusCreated,
		Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		},
	)

}

func (cfg *apiConfig) middlewareGetAllChirps(w http.ResponseWriter, req *http.Request) {

	chirps, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		log.Printf("Failed to save Chirp in DB")

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return
	}

	data := []Chirp{}
	for _, c := range chirps {
		data = append(
			data,
			Chirp{
				ID:        c.ID,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
				Body:      c.Body,
				UserID:    c.UserID,
			},
		)
	}

	sendJSONResponse(
		w,
		http.StatusOK,
		data,
	)
}

func (cfg *apiConfig) middlewareGetChirpByID(w http.ResponseWriter, req *http.Request) {

	chirpID := req.PathValue("chirpID")
	if chirpID == "" {
		log.Printf("Missing chirpID")

		sendJSONResponse(
			w,
			http.StatusBadRequest,
			jsonErr{Error: "Missing chirpID"},
		)

		return
	}

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		log.Printf("Could not parse chirpID string into a valid UUID")

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("Could not parse chirpID string into a valid UUID: %v", err)},
		)

		return
	}

	chirp, err := cfg.db.GetChirpByID(
		req.Context(),
		chirpUUID,
	)
	if err != nil {
		log.Printf("Chirp not found")

		sendJSONResponse(
			w,
			http.StatusNotFound,
			jsonErr{Error: fmt.Sprintf("Chirp not found: %v", err)},
		)

		return
	}

	sendJSONResponse(
		w,
		http.StatusOK,
		Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		},
	)

}

//===========/api/chirps: POST===============

type createChirpParameters struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

//===========/api/users: POST===============

type createUserParameters struct {
	Email string `json:"email"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

//===========Error Handling===============

type jsonErr struct {
	Error string `json:"error"`
}
