package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/Aaron-json/WsSession/internal/router"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {

	// load env variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Panicln("Error 	loading .env file:", err)
	}
	// set up routes
	mux := chi.NewRouter()
	router.Router(mux) // sconteet up all the routes for the application

	port := os.Getenv("PORT")
	addr := "127.0.0.1"

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", addr, port))
	if err != nil {
		log.Panicln(err)
	}
	if os.Getenv("ENV") != "production" {
		log.Printf("Server listening")
	}

	if err := http.Serve(l, mux); err != nil {
		log.Panicln(err)
	}
}
