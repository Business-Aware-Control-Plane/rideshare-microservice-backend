package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	TripsCollection     = "trips"
	RideFaresCollection = "ride_fares"

	// TTL durations
	TripTTL     = 90 * 24 * time.Hour // 90 days — keep trip history for a reasonable window
	RideFareTTL = 30 * time.Minute    // 30 minutes — fare estimates become stale quickly
)

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI      string
	Database string
}

// NewMongoDefaultConfig creates a new MongoDB configuration from environment variables
func NewMongoDefaultConfig() *MongoConfig {
	return &MongoConfig{
		URI:      os.Getenv("MONGODB_URI"),
		Database: "final-year-project",
	}
}

// NewMongoClient creates a new MongoDB client
func NewMongoClient(ctx context.Context, cfg *MongoConfig) (*mongo.Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongodb URI is required")
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("mongodb database is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	log.Printf("Successfully connected to MongoDB at %s", cfg.URI)
	return client, nil
}

// GetDatabase returns a database instance
func GetDatabase(client *mongo.Client, cfg *MongoConfig) *mongo.Database {
	return client.Database(cfg.Database)
}

// EnsureIndexes creates TTL indexes for all collections.
// Safe to call on every startup — MongoDB is a no-op if the index already exists.
func EnsureIndexes(ctx context.Context, database *mongo.Database) error {
	type indexSpec struct {
		collection string
		field      string
		ttl        time.Duration
	}

	specs := []indexSpec{
		{
			collection: TripsCollection,
			field:      "createdAt",
			ttl:        TripTTL,
		},
		{
			collection: RideFaresCollection,
			field:      "createdAt",
			ttl:        RideFareTTL,
		},
	}

	for _, spec := range specs {
		expireAfterSeconds := int32(spec.ttl.Seconds())
		model := mongo.IndexModel{
			Keys:    bson.D{{Key: spec.field, Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(expireAfterSeconds),
		}

		_, err := database.Collection(spec.collection).Indexes().CreateOne(ctx, model)
		if err != nil {
			return fmt.Errorf("failed to create TTL index on %s.%s: %w", spec.collection, spec.field, err)
		}

		log.Printf("TTL index ensured: collection=%s field=%s expiry=%s", spec.collection, spec.field, spec.ttl)
	}

	return nil
}
