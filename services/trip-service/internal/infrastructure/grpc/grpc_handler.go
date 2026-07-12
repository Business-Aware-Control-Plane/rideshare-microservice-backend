package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service   domain.TripService
	publisher *events.TripEventPublisher
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		service:   service,
		publisher: publisher,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	pickupCoord := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}
	destinationCoord := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}
	userID := req.UserID

	route, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch route: %v", err)
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)

	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID, route)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "Failed to generate trip fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	// Fetch and validate fare
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "Failed to validate fare: %v", err)
	}

	// Call create trip
	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "Failed to create trip: %v", err)
	}

	// Add a comment at the end of function to publish an event on the async comms module
	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to publish trip created event: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
