package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/tomkalva/chirpy-web-server/internal/auth"
	"github.com/tomkalva/chirpy-web-server/internal/database"
)

func (apiCfg *apiConfig) handlerChirpsDelete(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	userId, err := auth.ValidateJWT(bearerToken, apiCfg.jwtsecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	path := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := apiCfg.dbQueries.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	}

	if chirp.UserID != userId {
		respondWithError(w, http.StatusForbidden, "You can only delete your own chirps", nil)
		return
	}

	err = apiCfg.dbQueries.DeleteChirp(r.Context(),
		database.DeleteChirpParams{
			ID:     chirp.ID,
			UserID: userId,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete chirp", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
