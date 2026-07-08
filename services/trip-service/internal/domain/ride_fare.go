package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID `json:"_id" bson:"_id"`
	UserID            string
	PackageSlug       string // ex: van, luxury, sedan
	TotalPriceInCents float64
}
