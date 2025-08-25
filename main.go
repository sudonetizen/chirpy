package main

import (
    "log"
    "net/http"
)

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(http.StatusText(http.StatusOK)))
}

func main() {
    mux := http.NewServeMux()
    mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
    mux.HandleFunc("/healthz",  handlerHealthz)
   
    srv := &http.Server {
        Addr: ":8080",
        Handler: mux,
    }

    log.Println("serving on port 8080")
    err := srv.ListenAndServe()
    if err != nil {log.Fatal(err)}
}
