package main

import (
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/tomkalva/chirpy-web-server/internal/database"
)

func (apiCfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {
	var chirps []database.Chirp
	var err error

	authorId := r.URL.Query().Get("author_id")
	if authorId == "" {
		chirps, err = apiCfg.dbQueries.RetrieveAllChirps(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
			return
		}
	} else {
		authorID, err := uuid.Parse(authorId)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
			return
		}

		chirps, err = apiCfg.dbQueries.RetrieveAllChirpsByAuthor(r.Context(), authorID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
			return
		}
	}

	chirpArray := make([]Chirp, 0, len(chirps))

	for _, chirp := range chirps {
		respBody := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		chirpArray = append(chirpArray, respBody)
	}

	sortOrder := r.URL.Query().Get("sort")

	if sortOrder == "desc" {
		sort.Slice(chirpArray, func(i, j int) bool { return chirpArray[i].CreatedAt.After(chirpArray[j].CreatedAt) })
	} else {
		sort.Slice(chirpArray, func(i, j int) bool { return chirpArray[i].CreatedAt.Before(chirpArray[j].CreatedAt) })
	}

	respondWithJSON(w, http.StatusOK, chirpArray)
}

func (apiCfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("chirpID")
	authorID, err := uuid.Parse(path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
		return
	}

	chirp, err := apiCfg.dbQueries.GetChirpByID(r.Context(), authorID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	}

	respBody := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}
