package main

import (
	"net/http"
	"time"

	"github.com/tomkalva/chirpy-web-server/internal/auth"
)

func (apiCfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Token string `json:"token"`
	}

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting bearer token", err)
		return
	}
	user, err := apiCfg.dbQueries.GetUserFromRefreshToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Refresh token doesn't exist, expired or revoked", err)
		return
	}

	expiresIn := time.Duration(3600) * time.Second

	jwtToken, err := auth.MakeJWT(user.ID, apiCfg.jwtsecret, expiresIn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create JWT", err)
		return
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Token: jwtToken,
	})
}

func (apiCfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting bearer token", err)
		return
	}

	err = apiCfg.dbQueries.RevokeRefreshToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
