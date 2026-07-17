package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

// pgTrip represents the DB schema for trips
type pgTrip struct {
	ID         string      `gorm:"primaryKey;type:varchar(36)"`
	UserID     string      `gorm:"index;type:varchar(255)"`
	Status     string      `gorm:"type:varchar(50)"`
	RideFareID string      `gorm:"index;type:varchar(36)"`
	RideFare   *pgRideFare `gorm:"foreignKey:RideFareID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Driver     []byte      `gorm:"type:jsonb"`
	CreatedAt  time.Time   `gorm:"type:timestamptz"`
}

// TableName overrides the table name for pgTrip to 'trips'
func (pgTrip) TableName() string {
	return "trips"
}

// pgRideFare represents the DB schema for ride fares
type pgRideFare struct {
	ID                string    `gorm:"primaryKey;type:varchar(36)"`
	UserID            string    `gorm:"index;type:varchar(255)"`
	PackageSlug       string    `gorm:"type:varchar(50)"`
	TotalPriceInCents float64
	Route             []byte    `gorm:"type:jsonb"`
	CreatedAt         time.Time `gorm:"type:timestamptz"`
}

// TableName overrides the table name for pgRideFare to 'ride_fares'
func (pgRideFare) TableName() string {
	return "ride_fares"
}

type postgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository creates a new Postgres repository instance
func NewPostgresRepository(db *gorm.DB) *postgresRepository {
	// Automatically run migrations when the repository is initialized
	if err := db.AutoMigrate(&pgRideFare{}, &pgTrip{}); err != nil {
		panic(fmt.Sprintf("Failed to run postgres auto-migrations: %v", err))
	}
	return &postgresRepository{db: db}
}

// Mapping functions from Domain models to PostgreSQL database models

func toPGTrip(trip *domain.TripModel) (*pgTrip, error) {
	if trip == nil {
		return nil, nil
	}

	var driverBytes []byte
	if trip.Driver != nil {
		var err error
		driverBytes, err = json.Marshal(trip.Driver)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal driver: %w", err)
		}
	} else {
		driverBytes = []byte("{}")
	}

	pgT := &pgTrip{
		ID:        trip.ID.Hex(),
		UserID:    trip.UserID,
		Status:    trip.Status,
		Driver:    driverBytes,
		CreatedAt: trip.CreatedAt,
	}

	if trip.RideFare != nil {
		pgT.RideFareID = trip.RideFare.ID.Hex()
		fare, err := toPGRideFare(trip.RideFare)
		if err != nil {
			return nil, err
		}
		pgT.RideFare = fare
	}

	return pgT, nil
}

func toPGRideFare(fare *domain.RideFareModel) (*pgRideFare, error) {
	if fare == nil {
		return nil, nil
	}

	var routeBytes []byte
	if fare.Route != nil {
		var err error
		routeBytes, err = json.Marshal(fare.Route)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal route: %w", err)
		}
	} else {
		routeBytes = []byte("{}")
	}

	return &pgRideFare{
		ID:                fare.ID.Hex(),
		UserID:            fare.UserID,
		PackageSlug:       fare.PackageSlug,
		TotalPriceInCents: fare.TotalPriceInCents,
		Route:             routeBytes,
		CreatedAt:         fare.CreatedAt,
	}, nil
}

// Mapping functions from PostgreSQL database models to Domain models

func toDomainTrip(pgT *pgTrip) (*domain.TripModel, error) {
	if pgT == nil {
		return nil, nil
	}

	tripID, err := primitive.ObjectIDFromHex(pgT.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid trip ID hex: %w", err)
	}

	var tripDriver *pb.TripDriver
	if len(pgT.Driver) > 0 {
		var driver pb.TripDriver
		if err := json.Unmarshal(pgT.Driver, &driver); err != nil {
			return nil, fmt.Errorf("failed to unmarshal driver: %w", err)
		}
		tripDriver = &driver
	}

	trip := &domain.TripModel{
		ID:        tripID,
		UserID:    pgT.UserID,
		Status:    pgT.Status,
		Driver:    tripDriver,
		CreatedAt: pgT.CreatedAt,
	}

	if pgT.RideFare != nil {
		fare, err := toDomainRideFare(pgT.RideFare)
		if err != nil {
			return nil, err
		}
		trip.RideFare = fare
	}

	return trip, nil
}

func toDomainRideFare(pgF *pgRideFare) (*domain.RideFareModel, error) {
	if pgF == nil {
		return nil, nil
	}

	fareID, err := primitive.ObjectIDFromHex(pgF.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid ride fare ID hex: %w", err)
	}

	var route *tripTypes.OsrmApiResponse
	if len(pgF.Route) > 0 {
		var parsedRoute tripTypes.OsrmApiResponse
		if err := json.Unmarshal(pgF.Route, &parsedRoute); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route: %w", err)
		}
		route = &parsedRoute
	}

	return &domain.RideFareModel{
		ID:                fareID,
		UserID:            pgF.UserID,
		PackageSlug:       pgF.PackageSlug,
		TotalPriceInCents: pgF.TotalPriceInCents,
		Route:             route,
		CreatedAt:         pgF.CreatedAt,
	}, nil
}

// Repository Methods Implementation

func (r *postgresRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	trip.CreatedAt = time.Now().UTC()
	pgT, err := toPGTrip(trip)
	if err != nil {
		return nil, err
	}
	pgT.RideFare = nil // Clear association to prevent GORM from attempting duplicate insertions of ride fares

	if err := r.db.WithContext(ctx).Create(pgT).Error; err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	return trip, nil
}

func (r *postgresRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	var pgT pgTrip
	err := r.db.WithContext(ctx).Preload("RideFare").First(&pgT, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get trip by ID: %w", err)
	}

	return toDomainTrip(&pgT)
}

func (r *postgresRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if driver != nil {
		tripDriver := &pb.TripDriver{
			Id:             driver.Id,
			Name:           driver.Name,
			CarPlate:       driver.CarPlate,
			ProfilePicture: driver.ProfilePicture,
		}
		driverBytes, err := json.Marshal(tripDriver)
		if err != nil {
			return fmt.Errorf("failed to marshal driver: %w", err)
		}
		updates["driver"] = driverBytes
	}

	result := r.db.WithContext(ctx).Model(&pgTrip{}).Where("id = ?", tripID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update trip: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("trip not found: %s", tripID)
	}

	return nil
}

func (r *postgresRepository) SaveRideFare(ctx context.Context, fare *domain.RideFareModel) error {
	fare.CreatedAt = time.Now().UTC()
	pgF, err := toPGRideFare(fare)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(pgF).Error; err != nil {
		return fmt.Errorf("failed to save ride fare: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	var pgF pgRideFare
	err := r.db.WithContext(ctx).First(&pgF, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get ride fare by ID: %w", err)
	}

	return toDomainRideFare(&pgF)
}
