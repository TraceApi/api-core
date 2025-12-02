/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ProductCategory defines the specific regulation schema to apply.
type ProductCategory string

const (
	CategoryBattery    ProductCategory = "BATTERY_INDUSTRIAL"
	CategoryTextile    ProductCategory = "TEXTILE_APPAREL"
	CategoryElectronic ProductCategory = "CONSUMER_ELECTRONIC"
)

// PassportStatus represents the lifecycle state of the record.
type PassportStatus string

const (
	StatusDraft     PassportStatus = "DRAFT"     // Manufacturer is still editing
	StatusPublished PassportStatus = "PUBLISHED" // Locked and live on the blockchain/S3
	StatusRevoked   PassportStatus = "REVOKED"   // Recalled or erroneous
)

type ContextKey string

const (
	ViewContextKey        ContextKey = "view_context"
	ViewContextRestricted string     = "restricted"
	ViewContextPublic     string     = "public"
)

// Passport is the "Master Envelope" that aligns with GS1 Digital Link.
// This struct is mapped to your main Postgres table.
type Passport struct {
	ID              uuid.UUID       `json:"passportId" db:"id"`
	ProductCategory ProductCategory `json:"productCategory" db:"product_category"`
	Status          PassportStatus  `json:"status" db:"status"`

	// Manufacturer Info (The "Economic Operator")
	ManufacturerID   string `json:"manufacturerId" db:"manufacturer_id"` // e.g., DUNS number or Internal Tenant ID
	ManufacturerName string `json:"manufacturerName" db:"manufacturer_name"`

	// The "Payload" is stored as raw JSONB in Postgres.
	// We do not unmarshal it until we know the Category.
	Attributes json.RawMessage `json:"attributes" db:"attributes"`

	// Metadata
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
	PublishedAt      *time.Time `json:"publishedAt,omitempty" db:"published_at"`
	ImmutabilityHash string     `json:"immutabilityHash,omitempty" db:"immutability_hash"` // SHA-256 of the Attributes when Published
	StorageLocation  string     `json:"storageLocation,omitempty" db:"storage_location"`   // S3 URL
}

// --- The Polymorphic Payloads ---

// BatteryAttributes maps strictly to EU Regulation 2023/1542.
type BatteryAttributes struct {
	Model           string  `json:"batteryModel"`
	SerialNumber    string  `json:"serialNumber"`
	Chemistry       string  `json:"chemistry"` // e.g., "LFP", "NMC"
	RatedCapacityAh float64 `json:"ratedCapacity"`
	WeightKg        float64 `json:"weight"`

	// Compliance Data
	CarbonFootprint CarbonFootprint `json:"carbonFootprint"`
	Materials       []Material      `json:"materialComposition"`
}

type CarbonFootprint struct {
	TotalKgCO2e       float64 `json:"totalCarbonFootprint"`
	ShareOfRenewables float64 `json:"shareOfRenewables"`
	DeclarationUrl    string  `json:"declarationUrl"`
}

type Material struct {
	Name            string  `json:"material"` // "Cobalt", "Lithium"
	MassPercent     float64 `json:"massPercentage"`
	RecycledPercent float64 `json:"recycledContentPercentage"`
}

// TextileAttributes maps to the ESPR Textile Delegated Act (Draft).
type TextileAttributes struct {
	GarmentType      string             `json:"garmentType"`
	FiberComposition []FiberComposition `json:"fiberComposition"`
	CareInstructions CareInstructions   `json:"careInstructions"`
}

type FiberComposition struct {
	FiberName  string  `json:"fiberName"` // "Cotton", "Polyester"
	Percentage float64 `json:"percentage"`
	IsRecycled bool    `json:"isRecycled"`
}

type CareInstructions struct {
	Washing string `json:"washing"` // "Machine 30C"
	Drying  string `json:"drying"`
}

// --- Validation Logic ---

// GetBatteryAttributes safely unmarshals the raw JSONB into the struct.
func (p *Passport) GetBatteryAttributes() (*BatteryAttributes, error) {
	if p.ProductCategory != CategoryBattery {
		return nil, errors.New("passport is not a battery")
	}
	var attrs BatteryAttributes
	if err := json.Unmarshal(p.Attributes, &attrs); err != nil {
		return nil, err
	}
	return &attrs, nil
}

// Validate ensures the generic passport fields are correct.
// (Detailed schema validation happens at the Service layer using JSON Schema).
func (p *Passport) Validate() error {
	if p.ID == uuid.Nil {
		return errors.New("passport ID is required")
	}
	if p.ManufacturerID == "" {
		return errors.New("manufacturer ID is required")
	}
	if len(p.Attributes) == 0 {
		return errors.New("attributes payload cannot be empty")
	}
	return nil
}
