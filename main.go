package main

import (
    "fmt"
    "log"
    "net/http"
    "sync/atomic"
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
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(fmt.Sprintf("Hits: %v\n", cfg.fileserverHits.Load())))
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

func main() {
    mux := http.NewServeMux()
    apiCfg := &apiConfig{fileserverHits: atomic.Int32{}} 
    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
    mux.HandleFunc("GET /healthz",  handlerHealthz)
    mux.HandleFunc("GET /metrics", apiCfg.handlerHits)
    mux.HandleFunc("POST /reset", apiCfg.handlerReset)
   
    srv := &http.Server {
        Addr: ":8080",
        Handler: mux,
    }

    log.Println("serving on port 8080")
    err := srv.ListenAndServe()
    if err != nil {log.Fatal(err)}
}
