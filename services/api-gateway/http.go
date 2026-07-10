package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	grpcclients "ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// validation
	if reqBody.UserID == "" {
		http.Error(w, "User ID is Required", http.StatusBadRequest)
		return
	}

	jsonBody, _ := json.Marshal(reqBody)
	reader := bytes.NewReader(jsonBody)

	tripService, err := grpcclients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tripService.Close()

	// tripService.Client.PreviewTrip()

	// TODO: Call Trip Service
	url := "http://trip-service:8083/preview"
	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		log.Println("Failed to connect to trip service")
		return
	}
	defer resp.Body.Close()

	var resBody any
	if err := json.NewDecoder(resp.Body).Decode(&resBody); err != nil {
		http.Error(w, "Failed to parse JSON data from trip service", http.StatusBadRequest)
		return
	}

	response := contracts.APIResponse{Data: resBody}
	writeJSON(w, http.StatusCreated, response)
}
