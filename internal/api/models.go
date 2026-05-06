package api

import (
	"encoding/json"
	"time"
)

type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalItems int `json:"total_items"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

type AppleAccount struct {
	ID           string     `json:"id"`
	OrgID        string     `json:"org_id"`
	Name         string     `json:"name"`
	IssuerID     string     `json:"issuer_id"`
	KeyID        string     `json:"key_id"`
	TeamID       *string    `json:"team_id,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	SyncStatus   string     `json:"sync_status"`
	SyncError    *string    `json:"sync_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AppleAccountCreateRequest struct {
	Name       string `json:"name"`
	IssuerID   string `json:"issuer_id"`
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
	TeamID     string `json:"team_id,omitempty"`
}

type Certificate struct {
	ID              string    `json:"id"`
	AppleAccountID  string    `json:"apple_account_id"`
	AppleID         string    `json:"apple_id"`
	SerialNumber    string    `json:"serial_number"`
	DisplayName     string    `json:"display_name"`
	CertificateType string    `json:"certificate_type"`
	ExpirationDate  time.Time `json:"expiration_date"`
	Status          string    `json:"status"`
	Platform        string    `json:"platform"`
	CSRID           *string   `json:"csr_id,omitempty"`
	HasPrivateKey   bool      `json:"has_private_key"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CertificateCreateRequest struct {
	AppleAccountID  string `json:"apple_account_id"`
	CSRID           string `json:"csr_id"`
	CertificateType string `json:"certificate_type"`
}

type CertificateP12Response struct {
	P12Base64 string `json:"p12_base64"`
	Password  string `json:"password"`
	Filename  string `json:"filename"`
}

type Profile struct {
	ID                  string    `json:"id"`
	AppleAccountID      string    `json:"apple_account_id"`
	AppleID             string    `json:"apple_id"`
	Name                string    `json:"name"`
	ProfileType         string    `json:"profile_type"`
	ExpirationDate      time.Time `json:"expiration_date"`
	Status              string    `json:"status"`
	Platform            string    `json:"platform"`
	IdentifierAppleID   *string   `json:"identifier_apple_id,omitempty"`
	CertificateAppleIDs []string  `json:"certificate_apple_ids,omitempty"`
	DeviceAppleIDs      []string  `json:"device_apple_ids,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProfileDownloadResponse struct {
	MobileProvisionBase64 string `json:"mobileprovision_base64"`
	Filename              string `json:"filename"`
}

type ProfileCreateRequest struct {
	AppleAccountID      string   `json:"apple_account_id"`
	Name                string   `json:"name"`
	ProfileType         string   `json:"profile_type"`
	IdentifierAppleID   string   `json:"identifier_apple_id,omitempty"`
	CertificateAppleIDs []string `json:"certificate_apple_ids,omitempty"`
	DeviceAppleIDs      []string `json:"device_apple_ids,omitempty"`
}

type Identifier struct {
	ID             string          `json:"id"`
	AppleAccountID string          `json:"apple_account_id"`
	AppleID        string          `json:"apple_id"`
	Identifier     string          `json:"identifier"`
	Name           string          `json:"name"`
	Platform       string          `json:"platform"`
	IdentifierType string          `json:"identifier_type,omitempty"`
	Capabilities   json.RawMessage `json:"capabilities"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CapabilityRequest struct {
	CapabilityType string `json:"capability_type"`
}

type IdentifierCreateRequest struct {
	AppleAccountID string              `json:"apple_account_id"`
	Identifier     string              `json:"identifier"`
	Name           string              `json:"name"`
	Platform       string              `json:"platform"`
	IdentifierType string              `json:"identifier_type"`
	Capabilities   []CapabilityRequest `json:"capabilities"`
}

type Device struct {
	ID             string     `json:"id"`
	AppleAccountID string     `json:"apple_account_id"`
	AppleID        string     `json:"apple_id"`
	Name           string     `json:"name"`
	UDID           string     `json:"udid"`
	DeviceClass    string     `json:"device_class"`
	Model          *string    `json:"model,omitempty"`
	Platform       string     `json:"platform"`
	Status         string     `json:"status"`
	AddedDate      *time.Time `json:"added_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DeviceCreateRequest struct {
	AppleAccountID string `json:"apple_account_id"`
	Name           string `json:"name"`
	UDID           string `json:"udid"`
	Platform       string `json:"platform"`
}

type CSR struct {
	ID             string    `json:"id"`
	AppleAccountID string    `json:"apple_account_id"`
	Name           string    `json:"name"`
	Content        string    `json:"content"`
	KeyAlgorithm   string    `json:"key_algorithm,omitempty"`
	HasPrivateKey  bool      `json:"has_private_key"`
	CreatedAt      time.Time `json:"created_at"`
}

type CSRGenerateRequest struct {
	AppleAccountID string `json:"apple_account_id"`
	Name           string `json:"name"`
}

type CSRUploadRequest struct {
	AppleAccountID string `json:"apple_account_id"`
	Name           string `json:"name"`
	Content        string `json:"content"`
}

type DashboardSummary struct {
	Certificates struct {
		Total        int `json:"total"`
		Valid        int `json:"valid"`
		ExpiringSoon int `json:"expiring_soon"`
		Expired      int `json:"expired"`
	} `json:"certificates"`
	Profiles struct {
		Total        int `json:"total"`
		Active       int `json:"active"`
		ExpiringSoon int `json:"expiring_soon"`
		Expired      int `json:"expired"`
		Invalid      int `json:"invalid"`
	} `json:"profiles"`
	Devices     struct{ Total int `json:"total"` } `json:"devices"`
	Identifiers struct{ Total int `json:"total"` } `json:"identifiers"`
}
