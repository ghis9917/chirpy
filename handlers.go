package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/ghis9917/chirpy/internal/auth"
	"github.com/ghis9917/chirpy/internal/database"
	"github.com/google/uuid"
)

func handleReadiness(w http.ResponseWriter, req *http.Request) {
	sendResponse(
		w,
		CONTENT_TYPE_PLAIN_TEXT,
		http.StatusOK,
		[]byte(http.StatusText(http.StatusOK)),
	)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})

}

func (cfg *apiConfig) handleGetMetrics(w http.ResponseWriter, req *http.Request) {

	sendResponse(
		w,
		CONTENT_TYPE_HTML,
		http.StatusOK,
		[]byte(fmt.Sprintf(METRICS_HTML, cfg.fileserverHits.Load())),
	)

}

func (cfg *apiConfig) handleResetMetrics(w http.ResponseWriter, req *http.Request) {

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

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, req *http.Request) {

	params, err := extractParams(createUserParameters{}, req)
	if err != nil {

		sendJSONResponse(
			w,
			http.StatusInternalServerError,
			jsonErr{Error: fmt.Sprintf("%s", err)},
		)

		return
	}

	hash, err := auth.HashPassword(params.Password)
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
		database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
		},
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
		createUserResponse{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	)

}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, req *http.Request) {

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	params, err := extractParams(createChirpParameters{}, req)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	userID, err := auth.ValidateJWT(bearer, cfg.serverSecret)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	if len(params.Body) > VALID_CHIRP_LENGTH {
		sendJSONResponse(w, http.StatusBadRequest, jsonErr{Error: "Chirp is too long"})
		return
	}

	chirp, err := cfg.db.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   cleanChirp(params.Body),
			UserID: userID,
		},
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
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

func (cfg *apiConfig) handleGetAllChirps(w http.ResponseWriter, req *http.Request) {

	authorId := req.URL.Query().Get("author_id")
	sortParam := req.URL.Query().Get("sort")

	var chirps []database.Chirp
	var queryErr error
	if authorId == "" {
		chirps, queryErr = cfg.db.GetAllChirps(req.Context())
	} else {
		authorUUID, err := uuid.Parse(authorId)
		if err != nil {
			sendJSONResponse(w, http.StatusBadRequest, jsonErr{Error: fmt.Sprintf("Invalid UserID: %v", err)})
			return
		}
		chirps, queryErr = cfg.db.GetAllChirpsByAuthor(
			req.Context(),
			authorUUID,
		)
	}
	if queryErr != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", queryErr)})
		return
	}

	if sortParam == "desc" {
		slices.Reverse(chirps)
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

func (cfg *apiConfig) handleGetChirpByID(w http.ResponseWriter, req *http.Request) {

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

func (cfg *apiConfig) handleLogin(w http.ResponseWriter, req *http.Request) {

	params, err := extractParams(loginUserParameters{}, req)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	user, err := cfg.db.GetUserByEmail(
		req.Context(),
		params.Email,
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	check, err := auth.CheckPassword(params.Password, user.HashedPassword)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	if !check {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Incorrect password"})
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.serverSecret,
		time.Hour,
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	_, err = cfg.db.CreateRefreshRoken(
		req.Context(),
		database.CreateRefreshRokenParams{
			Token:  refreshToken,
			UserID: user.ID,
		},
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	sendJSONResponse(
		w,
		http.StatusOK,
		loginUserResponse{
			ID:           user.ID,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Email:        user.Email,
			IsChirpyRed:  user.IsChirpyRed,
			Token:        accessToken,
			RefreshToken: refreshToken,
		},
	)

}

func (cfg *apiConfig) handleRefresh(w http.ResponseWriter, req *http.Request) {

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: "Could not find bearer token"})
		return
	}

	token, err := cfg.db.GetRefreshToken(req.Context(), bearer)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Token not found"})
		return
	}
	if token.RevokedAt.Valid || time.Now().After(token.ExpiresAt) {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Token has been revoked"})
		return
	}

	accessToken, err := auth.MakeJWT(
		token.UserID,
		cfg.serverSecret,
		time.Hour,
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	sendJSONResponse(
		w,
		http.StatusOK,
		refreshReponse{
			Token: accessToken,
		},
	)

}

func (cfg *apiConfig) handleRevoke(w http.ResponseWriter, req *http.Request) {

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: "Could not find bearer token"})
		return
	}

	token, err := cfg.db.GetRefreshToken(req.Context(), bearer)
	if err != nil {
		sendJSONResponse(w, http.StatusNotFound, jsonErr{Error: "Token not found"})
		return
	}
	if token.RevokedAt.Valid && time.Now().After(token.ExpiresAt) {
		sendJSONResponse(w, http.StatusNotFound, jsonErr{Error: "Token has already been revoked or has expired"})
		return
	}

	if err = cfg.db.RevokeRefreshToken(
		req.Context(),
		database.RevokeRefreshTokenParams{
			RevokedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			Token: token.Token,
		},
	); err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: "Could not find bearer token"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, req *http.Request) {

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Could not find bearer token"})
		return
	}

	userID, err := auth.ValidateJWT(bearer, cfg.serverSecret)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	params, err := extractParams(updateUserParameters{}, req)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	user, err := cfg.db.UpdateUser(
		req.Context(),
		database.UpdateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
			ID:             userID,
		},
	)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return

	}

	sendJSONResponse(
		w,
		http.StatusOK,
		updateUserResponse{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	)

}

func (cfg *apiConfig) handleDeleteChirpByID(w http.ResponseWriter, req *http.Request) {

	chirpID := req.PathValue("chirpID")
	if chirpID == "" {
		sendJSONResponse(w, http.StatusBadRequest, jsonErr{Error: "Missing chirpID"})
		return
	}

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, jsonErr{Error: fmt.Sprintf("Invalid ChirpID: %v", err)})
		return
	}

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Could not find bearer token"})
		return
	}

	userID, err := auth.ValidateJWT(bearer, cfg.serverSecret)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	chirp, err := cfg.db.GetChirpByID(
		req.Context(),
		chirpUUID,
	)
	if err != nil {
		sendJSONResponse(w, http.StatusNotFound, jsonErr{Error: fmt.Sprintf("Chirp not found: %v", err)})
		return
	}

	if chirp.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err = cfg.db.DeleteChirpByID(
		req.Context(),
		chirpUUID,
	); err != nil {
		sendJSONResponse(w, http.StatusNotFound, jsonErr{Error: fmt.Sprintf("Chirp not found: %v", err)})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handleUpgradeUser(w http.ResponseWriter, req *http.Request) {

	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, jsonErr{Error: "Could not find apikey"})
		return
	}

	if apiKey != cfg.polkaSecret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params, err := extractParams(upgradeUserParams{}, req)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, jsonErr{Error: fmt.Sprintf("%s", err)})
		return
	}

	if params.Event != WEBHOOKS_UPGRADE_EVENT {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, jsonErr{Error: fmt.Sprintf("Invalid UserID: %v", err)})
		return
	}

	if err = cfg.db.UpgradeUser(
		req.Context(),
		userID,
	); err != nil {
		sendJSONResponse(w, http.StatusNotFound, jsonErr{Error: fmt.Sprintf("Couldn't find UserID: %v", err)})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
