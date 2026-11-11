package main

import (
	"context"
	"github.com/lambdawalker/site.card-designer/backend/internal/api"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	h, e := api.New(context.Background())
	if e != nil {
		log.Fatal(e)
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	s := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Fatal(s.ListenAndServe())
}
