package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Aaron-json/WsSession/internal/controllers"
	"github.com/joho/godotenv"
)

func main() {
	// load env variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Panicln("Error loading .env file:", err)
	}
	// set up routes
	http.DefaultServeMux.HandleFunc("GET /new-session/{sessionName}", controllers.CreateNewSession)
	http.DefaultServeMux.HandleFunc("GET /join-session/{sessionID}", controllers.JoinSession)
	port := os.Getenv("PORT")
	addr := "127.0.0.1"
	s := &http.Server{
		Addr: fmt.Sprint(addr, ":", port),
	}
	if os.Getenv("ENV") != "production" {
		log.Printf("Server starting")
	}
	if err := s.ListenAndServe(); err != nil {
		log.Panicln(err)
	}
}
