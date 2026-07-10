package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type previewTripRequest struct {
	UserID      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (req *previewTripRequest) toProto() *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID: req.UserID,
		StartLocation: &pb.Coordinate{
			Latitude:  req.Pickup.Latitude,
			Longitude: req.Pickup.Longitude,
		},
		EndLocation: &pb.Coordinate{
			Latitude:  req.Destination.Latitude,
			Longitude: req.Destination.Longitude,
		},
	}
}
