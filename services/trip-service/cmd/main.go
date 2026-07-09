package main

import (
	"log"
	"net/http"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
)

func main() {
	// ctx := context.Background()

	inmemRepo := repository.NewInmemRepository()
	svc := service.NewService(inmemRepo)
	mux := http.NewServeMux()

	httpHandler := h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", httpHandler.HandleTripPreview)

	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	log.Printf("Listening on %s\n", "")
	if err := server.ListenAndServe(); err != nil {
		log.Printf("server failed to start: %v\n", err)
	}
}
