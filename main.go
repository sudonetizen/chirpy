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
    "github.com/sudonetizen/auth"
    "github.com/sudonetizen/database"
)

// fileserverHits struct 
type apiConfig struct {
    fileserverHits atomic.Int32
    db *database.Queries
    tks string
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
type onlyEmail struct {
    Email string `json:"email"`
}

type email struct {
    Password string        `json:"password"`
    Email    string        `json:"email"`
    Expires  time.Duration `json:"expires_in_seconds"`
}

type user struct {
    Id         uuid.UUID `json:"id"`
    Created_at time.Time `json:"created_at"`
    Updated_at time.Time `json:"updated_at"`
    Email      string    `json:"email"`
    Token      string    `json:"token"`
    RToken     string    `json:"refresh_token"`
}

// token struct
type tokenStruct struct {
    Token string `json:"token"`
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
        log.Printf("error with deleting users: %v\n", err)
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
        w.WriteHeader(400)
        return
    } 

    // getting token from header
    ss, err := auth.GetBearerToken(r.Header)
    if err != nil {
        log.Printf("error with getting token: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // validating JWT 
    userid, err := auth.ValidateJWT(ss, cfg.tks)
    if err != nil {
        log.Printf("error with validating jwt: %v\n", err)
        w.WriteHeader(401)
        return
    }

    msg.User_id = userid

    // checking user_id 
    _, err = cfg.db.GetUser(r.Context(), msg.User_id)
    if err != nil {
        log.Printf("smth: %v\n", msg.User_id)
        log.Printf("error with 1111 checking user_id: %v\n", err)
        w.WriteHeader(400)
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

    // creating a chirp response 
    chrp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{msgString, msg.User_id})
    if err != nil {
        log.Printf("error with creating chirp: %v\n", err)
        w.WriteHeader(500)
        return 
    }

    // encoding response 
    chirpRes := chirp_res{Id: chrp.ID, Created_at: chrp.CreatedAt, Updated_at: chrp.UpdatedAt, Body: chrp.Body, User_id: chrp.UserID}
    data, err := json.Marshal(chirpRes)
    if err != nil {
        log.Printf("error with creating res json: %v\n", err)
        w.WriteHeader(500)
        return 
    }
    
    // sending response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(201)
    w.Write(data)
}

// handles -> post /api/users
func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
    // decoding email and password into struct 
    eml := email{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&eml)

    if err != nil {
        log.Printf("error with decoding: %v\n", err)
        w.WriteHeader(400)
        return
    } 

    // hashing password 
    hash, err := auth.HashPassword(eml.Password)
    
    if err != nil {
        log.Printf("error with hashing: %v\n", err)
        w.WriteHeader(500)
        return 
    }
    
    // creating user
    usr, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{eml.Email, hash})
      
    if err != nil {
        log.Printf("error with creating user: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // encoding response 
    res := user{Id: usr.ID, Created_at: usr.CreatedAt, Updated_at: usr.UpdatedAt, Email: usr.Email}
    data, err := json.Marshal(res)
    
    if err != nil {
        log.Printf("error with creating json: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // sending response 
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

// handles -> post /api/login
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
    // decoding password and email from request into struct
    eml := email{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&eml)

    if err != nil {
        log.Printf("error with decoding: %v\n", err)
        w.WriteHeader(400)
        return
    }

    // checking expires_in_seconds 
    if eml.Expires == 0 {eml.Expires = time.Duration(3600 * time.Second)}

    // getting user by email  
    usr, err := cfg.db.GetUserByEml(r.Context(), eml.Email)
    
    if err != nil {
        log.Printf("error with getting user by email: %v\n", err)
        w.WriteHeader(401)
        w.Write([]byte("incorrect email"))
        return 
    }

    // checking password 
    err = auth.CheckPasswordHash(eml.Password, usr.HashedPassword)
    
    if err != nil {
        log.Printf("error with checking password hash: %v\n", err)
        w.WriteHeader(401)
        w.Write([]byte("incorrect password"))
        return 
    }

    // creating token 
    tokenU, err := auth.MakeJWT(usr.ID, cfg.tks, eml.Expires)

    if err != nil {
        log.Printf("error with creating token: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // creating refresh token
    rtkn, err := auth.MakeRefreshToken()
    
    if err != nil {
        log.Printf("error with creating refresh token: %v\n", err)
        w.WriteHeader(500)
        return 
    }

    // saving refresh token to database 
    _, err = cfg.db.CreateRToken(r.Context(), database.CreateRTokenParams{rtkn, usr.ID, time.Now().Add((60*24*3600) * time.Second)})

    if err != nil {
        log.Printf("error with saving rtoken: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // encoding response 
    resp := user{usr.ID, usr.CreatedAt, usr.UpdatedAt, usr.Email, tokenU, rtkn}
    data, err := json.Marshal(resp) 

    if err != nil {
        log.Printf("error with encoding json: %v\n", err)
        w.WriteHeader(500)
        return
    }
    
    // sending response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200) 
    w.Write(data)
}

// handles -> post /api/refresh 
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
    // getting token 
    tkn, err := auth.GetBearerToken(r.Header)
    
    if err != nil {
        log.Printf("error with getting token: %v\n", err)
        w.WriteHeader(500)
        return
    }
    
    // searching token in database and checking expire time for nil 
    rtkn, err := cfg.db.GetRToken(r.Context(), tkn)
    
    if err != nil {
        log.Printf("error with searching token: %v\n", err)
        w.WriteHeader(401)
        return
    }

    nullTime := time.Time{}
    if rtkn.RevokedAt.Time != nullTime {
        log.Println("error with revoked time")
        w.WriteHeader(401)
        return
    }
    
    // creating new token  
    ss, err := auth.MakeJWT(rtkn.UserID, cfg.tks, time.Duration(3600 * time.Second))

    if err != nil {
        log.Printf("error with creating token: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // encoding response 
    resp := tokenStruct{Token: ss}
    data, err := json.Marshal(resp) 
    
    if err != nil {
        log.Printf("error with marshalling json: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // sending response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200) 
    w.Write(data)
}

// handles -> post /api/revoke 
func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
    // getting token 
    tkn, err := auth.GetBearerToken(r.Header)
    
    if err != nil {
        log.Printf("error with getting token: %v\n", err)
        w.WriteHeader(500)
        return
    }
    
    // updating refresh token's updated_at and revoked_at timestamps 
    err = cfg.db.UpdateRToken(r.Context(), tkn)  

    if err != nil {
        log.Printf("error with updating token: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // response 
    w.WriteHeader(204) 
}

// handles -> put /api/users 
func (cfg *apiConfig) handlerUUpdate(w http.ResponseWriter, r *http.Request) {
    // getting token 
    tkn, err := auth.GetBearerToken(r.Header)
    
    if err != nil {
        log.Printf("476 error with getting token: %v\n", err)
        w.WriteHeader(401)
        return
    }

    // validating JWT 
    userid, err := auth.ValidateJWT(tkn, cfg.tks)
    if err != nil {
        log.Printf("484 error with validating jwt: %v\n", err)
        w.WriteHeader(401)
        return
    }

    // decoding request 
    eml := email{}
    decoder := json.NewDecoder(r.Body)
    err = decoder.Decode(&eml)
    
    if err != nil {
        log.Printf("495 error with decoding request: %v\n", err)
        w.WriteHeader(401)
        return
    }

    // hashing password 
    hashed, err := auth.HashPassword(eml.Password)
 
    if err != nil {
        log.Printf("504 error with hashing password: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // updating user's email and password at database
    err = cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{eml.Email, hashed, userid})
    
    if err != nil {
        log.Printf("512 error with updating user: %v\n", err)
        w.WriteHeader(401)
        return
    }

    // encoding response
    res := onlyEmail{eml.Email}
    data, err := json.Marshal(res)

    if err != nil {
        log.Printf("522 error with marshalling json: %v\n", err)
        w.WriteHeader(500)
        return
    }

    // sending response 
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200) 
    w.Write(data)
}

// handles -> delete /api/chirps/{chirpID}
func (cfg *apiConfig) handlerDelChirp(w http.ResponseWriter, r *http.Request) {
    // getting token 
    tkn, err := auth.GetBearerToken(r.Header)
    
    if err != nil {
        log.Printf("error with getting token: %v\n", err)
        w.WriteHeader(401)
        return
    }

    // validating JWT 
    userid, err := auth.ValidateJWT(tkn, cfg.tks)
    if err != nil {
        log.Printf("error with validating jwt: %v\n", err)
        w.WriteHeader(401)
        return
    }

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
    
    // checking user match 
    if userid != chp.UserID {
        log.Println("error with user match")
        w.WriteHeader(403)
        return 
    }

    // deleting chirp 
    err = cfg.db.DelChirp(r.Context(), chp.ID)
    
    if err != nil {
        log.Printf("error with deleting chirp: %v\n", err)
        w.WriteHeader(500)
        return
    } 

    // response
    w.WriteHeader(204)
    
}

func main() {
    // get DB_URL
    godotenv.Load()
    dbURL := os.Getenv("DB_URL")
    tknS := os.Getenv("SECRET")
    if tknS == "" {log.Fatal("secret is not set")}
    // connection to database
    db, err := sql.Open("postgres", dbURL)
    dbQueries := database.New(db)

    mux := http.NewServeMux()
    apiCfg := &apiConfig{fileserverHits: atomic.Int32{}, db: dbQueries, tks: tknS} 

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
    mux.HandleFunc("GET /api/healthz",  handlerHealthz)

    mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDelChirp)
    mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
    mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
    mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)

    mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
    mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
    mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
    mux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
    mux.HandleFunc("PUT /api/users", apiCfg.handlerUUpdate)

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
