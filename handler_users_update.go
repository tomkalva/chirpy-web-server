package main

import (
	"encoding/json"
	"net/http"

	"github.com/tomkalva/chirpy-web-server/internal/auth"
	"github.com/tomkalva/chirpy-web-server/internal/database"
)

func (apiCfg *apiConfig) handlerUsersUpdate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	id, err := auth.ValidateJWT(bearerToken, apiCfg.jwtsecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}

	updatedUser, err := apiCfg.dbQueries.UpdateUserEmailAndPassword(r.Context(),
		database.UpdateUserEmailAndPasswordParams{
			Email:          params.Email,
			HashedPassword: hashedPassword,
			ID:             id,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update user", err)
		return
	}

	respBody := User{
		ID:          updatedUser.ID,
		CreatedAt:   updatedUser.CreatedAt,
		UpdatedAt:   updatedUser.UpdatedAt,
		Email:       updatedUser.Email,
		IsChirpyRed: updatedUser.IsChirpyRed,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}
