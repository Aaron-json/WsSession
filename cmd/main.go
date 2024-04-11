package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Aaron-json/WsSession/internal/router"
	"github.com/joho/godotenv"
)

func main() {
	// load env variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Panicln("Error loading .env file:", err)
	}
	// set up routes
	mux := router.NewRouter()

	port := os.Getenv("PORT")
	addr := "127.0.0.1"

	s := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprint(addr, ":", port),
	}
	if os.Getenv("ENV") != "production" {
		log.Printf("Server starting")
	}
	if err := s.ListenAndServe(); err != nil {
		log.Panicln(err)
	}
}
