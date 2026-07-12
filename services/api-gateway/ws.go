package main

import (
	"log"
	"net/http"
	grpcclients "ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userId := r.URL.Query().Get("userID")
	if userId == "" {
		log.Println("No UserId Provided")
		return
	}

	// Add connection to manager
	connManager.Add(userId, conn)
	defer connManager.Remove(userId)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error Reading the Message: %v", err)
			break
		}

		log.Printf("Received Message: %s", msg)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userId := r.URL.Query().Get("userID")
	if userId == "" {
		log.Println("No UserId Provided")
		return
	}

	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("No Package Slug Provided")
		return
	}

	// Add connection to manager
	connManager.Add(userId, conn)

	ctx := r.Context()

	driverService, err := grpcclients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Closing connections
	defer func() {
		connManager.Remove(userId)

		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userId,
			PackageSlug: packageSlug,
		})

		driverService.Close()

		log.Println("Driver unregistered: ", userId)
	}()

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userId,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error registering driver: %v", err)
		return
	}

	if err := connManager.SendMessage(userId, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}); err != nil {
		log.Printf("Error Sending Message: %v", err)
		return
	}

	// Initialize queue consumers
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error Reading the Message: %v", err)
			break
		}

		log.Printf("Received Message: %s", msg)
	}

}
