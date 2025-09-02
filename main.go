package main

import (
    "os"
    "fmt"
    "log"
    "time"
    "strings"
    "net/http"
    "sync/atomic"
    "encoding/json"
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
    "github.com/google/uuid"
    "github.com/sudonetizen/database"
)

// fileserverHits struct 
type apiConfig struct {
    fileserverHits atomic.Int32
    db *database.Queries
}

// chirp structs 
type chirp struct {
    Body    string `json:"body"`
    User_id uuid.UUID `json:"user_id"`
}

type chirp_res struct {
    Id         uuid.UUID `json:"id"`
    Created_at time.Time `json:"created_at"`
    Updated_at time.Time `json:"updated_at"`
    Body       string    `json:"body"`
    User_id    uuid.UUID `json:"user_id"`
}

type ch_res struct {
    BodyClean string `json:"cleaned_body"`
}

type ch_err struct {
    Error string `json:"error"`
}

type ch_vld struct {
    Valid bool `json:"valid"`
}

// user structs
type email struct {
    Email string `json:"email"`
}

type user struct {
    Id         uuid.UUID `json:"id"`
    Created_at time.Time `json:"created_at"`
    Updated_at time.Time `json:"updated_at"`
    Email      string    `json:"email"`
}

// middleware that count fileserver hits by using on handler function 
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
        cfg.fileserverHits.Add(1)
        next.ServeHTTP(w, r)
    }) 
}

// shows hits of fileserver
func (cfg *apiConfig) handlerHits(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(fmt.Sprintf(hits_templ, cfg.fileserverHits.Load())))
}

// handles -> post /admin/reset 
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(fmt.Sprintln("reset done, hits now 0")))

    err := cfg.db.DeleteUsers(r.Context())
    if err != nil {
        log.Printf("error with marshalling ch_err: %v\n", err)
        w.WriteHeader(500)
    }

    log.Printf("deleted users")
    
}

// for checking health of web app
func handlerHealthz(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(http.StatusText(http.StatusOK)))
}

// handles -> post /api/chirps 
func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
    // decoding chirp message
    msg := chirp{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&msg)

    if err != nil {
        log.Printf("error with decoding: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // checking user_id 
    _, err = cfg.db.GetUser(r.Context(), msg.User_id)
    if err != nil {
        log.Printf("error with checking user_id: %v\n", err)
        w.WriteHeader(500)
        return 
        }

    // checking for length
    if len(msg.Body) > 140 {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(400)

        err_msg := ch_err{Error: "Chirp is too long"}
        data, err := json.Marshal(err_msg)
        if err != nil {
            log.Printf("error with marshalling ch_err: %v\n", err)
            w.WriteHeader(500)
            return 
        }

        w.Write(data) 
        return
    }

    // cleaning chirp message
    msgString := msg.Body
    profane_words := map[string]struct{}{"kerfuffle": {}, "sharbert": {}, "fornax": {}}

    for _, word := range strings.Fields(msgString) {
        _, ok := profane_words[strings.ToLower(word)]
        if ok { msgString = strings.ReplaceAll(msgString, word, "****") }
    }

    // creating a chirp and sending response 
    chrp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{msgString, msg.User_id})
    if err != nil {
        log.Printf("error with creating chirp: %v\n", err)
        w.WriteHeader(500)
        return 
    }

    chirpRes := chirp_res{Id: chrp.ID, Created_at: chrp.CreatedAt, Updated_at: chrp.UpdatedAt, Body: chrp.Body, User_id: chrp.UserID}
    data, err := json.Marshal(chirpRes)
    if err != nil {
        log.Printf("error with creating res json: %v\n", err)
        w.WriteHeader(500)
        return 
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(201)
    w.Write(data)
}

// handles -> post /api/users
func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
    // decoding email struct 
    eml := email{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&eml)

    if err != nil {
        log.Printf("error with decoding: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // creating user
    usr, err := cfg.db.CreateUser(r.Context(), eml.Email)
      
    if err != nil {
        log.Printf("error with creating user: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // sending response 
    res := user{Id: usr.ID, Created_at: usr.CreatedAt, Updated_at: usr.UpdatedAt, Email: usr.Email}
    data, err := json.Marshal(res)
    
    if err != nil {
        log.Printf("error with creating json: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    w.Header().Set("Content-Type", "application/json") 
    w.WriteHeader(201)
    w.Write(data)
}

// handles -> get /api/chirps
func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
    // getting chirps
    chirps, err := cfg.db.GetChirps(r.Context())

    if err != nil {
        log.Printf("error with getting chirps: %v\n", err)
        w.WriteHeader(500)
        return
    } 
    
    // encoding chirps 
    chirps_list := []chirp_res{} 
    for _, ch := range chirps {
        chch := chirp_res{ch.ID, ch.CreatedAt, ch.UpdatedAt, ch.Body, ch.UserID}
        chirps_list = append(chirps_list, chch)
    }    

    data, err := json.Marshal(chirps_list)
    
    if err != nil {
        log.Printf("error with marshalling chirps: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // sending response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200) 
    w.Write(data)
}

// handles -> get /api/chirps/chirpID 
func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
    // getting chirp 
    id, err := uuid.Parse(r.PathValue("chirpID"))
    if err != nil {
        log.Printf("error with parsing uuid: %v\n", err) 
        w.WriteHeader(400)
        w.Write([]byte("chirp id is invalid"))
        return
    }

    chp, err := cfg.db.GetChirp(r.Context(), id)

    if err != nil {
        log.Printf("error with getting chirp: %v\n", err)
        w.WriteHeader(404)
        return
    } 
    
    // encoding 
    chch := chirp_res{chp.ID, chp.CreatedAt, chp.UpdatedAt, chp.Body, chp.UserID}
    dta, err := json.Marshal(chch)

    if err != nil {
        log.Printf("error with marshalling chirp: %v\n", err)
        w.WriteHeader(500)
        return
    } 
 
    // response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200) 
    w.Write(dta)
}

func main() {
    // get DB_URL
    godotenv.Load()
    dbURL := os.Getenv("DB_URL")
    // connection to database
    db, err := sql.Open("postgres", dbURL)
    dbQueries := database.New(db)

    mux := http.NewServeMux()
    apiCfg := &apiConfig{fileserverHits: atomic.Int32{}, db: dbQueries} 

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
    mux.HandleFunc("GET /api/healthz",  handlerHealthz)

    mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
    mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
    mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)

    mux.HandleFunc("POST /api/users", apiCfg.handlerUsers)

    mux.HandleFunc("GET /admin/metrics", apiCfg.handlerHits)
    mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
   
    srv := &http.Server {
        Addr: ":8080",
        Handler: mux,
    }

    log.Println("serving on port 8080")
    err = srv.ListenAndServe()
    if err != nil {log.Fatal(err)}
}
