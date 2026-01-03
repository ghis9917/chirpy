package main

import (
	"sync/atomic"
	"time"

	"github.com/ghis9917/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	serverSecret   string
	polkaSecret    string
}

//===========/api/chirps: POST===============

type createChirpParameters struct {
	Body string `json:"body"`
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
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createUserResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

//===========/api/login: POST===============

type loginUserParameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginUserResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

//===========/api/refresh: POST===============

type refreshReponse struct {
	Token string `json:"token"`
}

//===========/api/users: POST===============

type updateUserParameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateUserResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

//===========/api/polka/webhooks: POST===============

type upgradeUserParams struct {
	Event string                `json:"event"`
	Data  upgradeUserParamsData `json:"data"`
}

type upgradeUserParamsData struct {
	UserID string `json:"user_id"`
}

//===========Error Handling===============

type jsonErr struct {
	Error string `json:"error"`
}
