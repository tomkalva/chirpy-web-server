package main

import "net/http"

func (apiCfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if apiCfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Reset is only allowed in dev environment", nil)
		return
	}

	err := apiCfg.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete users", err)
		return
	}

	apiCfg.fileserverHits.Store(0)
	w.Write([]byte("OK"))
}
