package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	math "math/rand/v2"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/util"

	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		rdb: rdb,
	}
}

func (s *Service) FindAvailableDrivers(ctx context.Context, packageType string) []string {
	driverIds, err := s.rdb.SMembers(ctx, "drivers:package:"+packageType).Result()
	if err != nil {
		log.Printf("Failed to find available drivers from Redis: %v", err)
		return []string{}
	}

	if len(driverIds) == 0 {
		return []string{}
	}

	return driverIds
}

func (s *Service) RegisterDriver(ctx context.Context, driverId string, packageSlug string) (*pb.Driver, error) {
	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]
	randomAvatar := util.GetRandomAvatar(1)
	randomPlate := GenerateRandomPlate()

	// we can ignore this property for now, but it must be sent to the frontend.
	geohashStr := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohashStr,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

	data, err := json.Marshal(driver)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal driver info: %w", err)
	}

	// Store driver info in Redis hash
	err = s.rdb.HSet(ctx, "drivers:info", driverId, data).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to save driver info to Redis: %w", err)
	}

	// Add driver to the package set in Redis
	err = s.rdb.SAdd(ctx, "drivers:package:"+packageSlug, driverId).Err()
	if err != nil {
		// Clean up the hash if set insertion fails
		_ = s.rdb.HDel(ctx, "drivers:info", driverId)
		return nil, fmt.Errorf("failed to save driver package set to Redis: %w", err)
	}

	return driver, nil
}

func (s *Service) UnregisterDriver(ctx context.Context, driverId string) {
	// Retrieve driver info to know their package slug
	data, err := s.rdb.HGet(ctx, "drivers:info", driverId).Result()
	if err == redis.Nil {
		// Driver not registered/already unregistered
		return
	} else if err != nil {
		log.Printf("Failed to get driver info for unregistration: %v", err)
		return
	}

	var driver pb.Driver
	if err := json.Unmarshal([]byte(data), &driver); err != nil {
		log.Printf("Failed to unmarshal driver info: %v", err)
		return
	}

	// Remove from package set in Redis
	err = s.rdb.SRem(ctx, "drivers:package:"+driver.PackageSlug, driverId).Err()
	if err != nil {
		log.Printf("Failed to remove driver from package set: %v", err)
	}

	// Delete from driver info hash in Redis
	err = s.rdb.HDel(ctx, "drivers:info", driverId).Err()
	if err != nil {
		log.Printf("Failed to delete driver info from Redis: %v", err)
	}
}
