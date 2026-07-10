package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/tomkalva/chirpy-web-server/internal/auth"
	"github.com/tomkalva/chirpy-web-server/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwtsecret      string
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func replaceBadWords(body string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}

	words := strings.Split(body, " ")
	for i, word := range words {
		for _, badWord := range badWords {
			if strings.ToLower(word) == badWord {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error opening DB: %s", err)
		return
	}
	dbQueries := database.New(db)
	platform := os.Getenv("PLATFORM")
	jwtsecret := os.Getenv("JWTSecret")

	apiCfg := apiConfig{
		dbQueries: dbQueries,
		platform:  platform,
		jwtsecret: jwtsecret,
	}

	const filepathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, apiCfg.fileserverHits.Load())))
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if apiCfg.platform != "dev" {
			w.WriteHeader(403)
			return
		}
		err := apiCfg.dbQueries.DeleteAllUsers(r.Context())
		if err != nil {
			log.Printf("Error with DeleteAllUsers: %s", err)
			return
		}

		apiCfg.fileserverHits.Store(0)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		type errorResponse struct {
			Error string `json:"error"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respBody := errorResponse{
				Error: "Error decoding parameters",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			log.Printf("Error hashing password: %s", err)
			return
		}

		user, err := apiCfg.dbQueries.CreateUser(r.Context(),
			database.CreateUserParams{
				Email:          params.Email,
				HashedPassword: hashedPassword,
			})
		if err != nil {
			log.Printf("Error creating user: %s", err)
			return
		}

		respBody := User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)
	})

	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		type errorResponse struct {
			Error string `json:"error"`
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respBody := errorResponse{
				Error: "Unauthorized",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		id, err := auth.ValidateJWT(bearerToken, apiCfg.jwtsecret)
		if err != nil {
			respBody := errorResponse{
				Error: "Unauthorized",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err = decoder.Decode(&params)
		if err != nil {
			respBody := errorResponse{
				Error: "Error decoding parameters",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			respBody := errorResponse{
				Error: "Error hashing password",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		updatedUser, err := apiCfg.dbQueries.UpdateUserEmailAndPassword(r.Context(),
			database.UpdateUserEmailAndPasswordParams{
				Email:          params.Email,
				HashedPassword: hashedPassword,
				ID:             id,
			})
		if err != nil {
			respBody := errorResponse{
				Error: "Error updateing user",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		respBody := User{
			ID:        updatedUser.ID,
			CreatedAt: updatedUser.CreatedAt,
			UpdatedAt: updatedUser.UpdatedAt,
			Email:     updatedUser.Email,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}

		type returnVals struct {
			CleanedBody string `json:"cleaned_body"`
		}

		type errorResponse struct {
			Error string `json:"error"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respBody := errorResponse{
				Error: "Error decoding parameters",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respBody := errorResponse{
				Error: "Unauthorized",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		id, err := auth.ValidateJWT(bearerToken, apiCfg.jwtsecret)
		if err != nil {
			respBody := errorResponse{
				Error: "Unauthorized",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		if len(params.Body) > 140 {
			respBody := errorResponse{
				Error: "Chirp is too long",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			w.Write(dat)

			return
		}

		cleanedString := replaceBadWords(params.Body)

		chirp, err := apiCfg.dbQueries.CreateChirp(r.Context(),
			database.CreateChirpParams{
				Body:   cleanedString,
				UserID: id,
			})
		if err != nil {
			log.Printf("Error creating chirp: %s", err)
			return
		}

		respBody := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := apiCfg.dbQueries.RetrieveAllChirps(r.Context())
		if err != nil {
			log.Printf("Error retrieving chirps: %s", err)
			return
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

		dat, err := json.Marshal(chirpArray)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		type errorResponse struct {
			Error string `json:"error"`
		}

		path := r.PathValue("chirpID")
		uuid, err := uuid.Parse(path)
		if err != nil {
			fmt.Println("Invalid UUID:", err)
			return
		}

		chirp, err := apiCfg.dbQueries.GetChirpByID(r.Context(), uuid)
		if err != nil {
			respBody := errorResponse{
				Error: "Chirp not found",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			w.Write(dat)
			return
		}

		respBody := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		type errorResponse struct {
			Error string `json:"error"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respBody := errorResponse{
				Error: "Error decoding parameters",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write(dat)

			return
		}

		user, err := apiCfg.dbQueries.GetUserByEmail(r.Context(), params.Email)
		if err != nil {
			respBody := errorResponse{
				Error: "Incorrect email or password",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
		if err != nil || match == false {
			respBody := errorResponse{
				Error: "Incorrect email or password",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		expiresIn := time.Duration(3600) * time.Second

		jwtToken, err := auth.MakeJWT(user.ID, apiCfg.jwtsecret, expiresIn)
		if err != nil {
			log.Printf("Error making jwt: %s", err)
			return
		}

		rToken, err := auth.MakeRefreshToken()
		if err != nil {
			log.Printf("Error making refreshToken: %s", err)
			return
		}

		refreshToken, err := apiCfg.dbQueries.CreateRefreshToken(r.Context(),
			database.CreateRefreshTokenParams{
				Token:     rToken,
				UserID:    user.ID,
				ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
			})
		if err != nil {
			log.Printf("Error creating refreshToken: %s", err)
			return
		}

		respBody := User{
			ID:           user.ID,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Email:        user.Email,
			Token:        jwtToken,
			RefreshToken: refreshToken.Token,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		type errorResponse struct {
			Error string `json:"error"`
		}

		type returnVals struct {
			Token string `json:"token"`
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respBody := errorResponse{
				Error: "Error getting bearerToken",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}
		user, err := apiCfg.dbQueries.GetUserFromRefreshToken(r.Context(), bearerToken)
		if err != nil {
			respBody := errorResponse{
				Error: "Refresh token doesn't exist, expired or revoked",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		expiresIn := time.Duration(3600) * time.Second

		jwtToken, err := auth.MakeJWT(user.ID, apiCfg.jwtsecret, expiresIn)
		if err != nil {
			log.Printf("Error making jwt: %s", err)
			return
		}

		respBody := returnVals{
			Token: jwtToken,
		}

		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		type errorResponse struct {
			Error string `json:"error"`
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respBody := errorResponse{
				Error: "Error getting bearerToken",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		err = apiCfg.dbQueries.RevokeRefreshToken(r.Context(), bearerToken)
		if err != nil {
			respBody := errorResponse{
				Error: "Error revokeing refreshToken",
			}

			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshaling: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write(dat)

			return
		}

		w.WriteHeader(204)
	})

	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
