package main

import (
    "fmt"
    "log"
    "strings"
    "net/http"
    "sync/atomic"
    "encoding/json"
)

type apiConfig struct {
    fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
        cfg.fileserverHits.Add(1)
        next.ServeHTTP(w, r)
    }) 
}

func (cfg *apiConfig) handlerHits(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(fmt.Sprintf(hits_templ, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(fmt.Sprintln("reset done, hits now 0")))
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(http.StatusText(http.StatusOK)))
}

type chirp struct {
    Body string `json:"body"`
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

func handlerJson(w http.ResponseWriter, r *http.Request) {
    msg := chirp{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&msg)

    if err != nil {
        log.Printf("error with decoding: %v\n", err)
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

    // sending response 
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)

    res_msg := ch_res{BodyClean: msgString}
    data, err := json.Marshal(res_msg)
    if err != nil {
        log.Printf("error with marshalling ch_err: %v\n", err)
        w.WriteHeader(500)
        return 
    }
    
    w.Write(data)
}

func main() {
    mux := http.NewServeMux()
    apiCfg := &apiConfig{fileserverHits: atomic.Int32{}} 
    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
    mux.HandleFunc("GET /api/healthz",  handlerHealthz)
    mux.HandleFunc("POST /api/validate_chirp", handlerJson)
    mux.HandleFunc("GET /admin/metrics", apiCfg.handlerHits)
    mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
   
    srv := &http.Server {
        Addr: ":8080",
        Handler: mux,
    }

    log.Println("serving on port 8080")
    err := srv.ListenAndServe()
    if err != nil {log.Fatal(err)}
}
