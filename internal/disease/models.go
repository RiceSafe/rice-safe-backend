package disease

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// InfoSection represents a detailed section (Title/Description)
type InfoSection struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// InfoSections is a helper type for JSONB scanning
type InfoSections []InfoSection

// Value implements driver.Valuer for JSONB
func (a InfoSections) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner for JSONB
func (a *InfoSections) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

// StringArray is a helper type for JSONB array of strings
type StringArray []string

// Value implements driver.Valuer for JSONB
func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner for JSONB
func (a *StringArray) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type Disease struct {
	ID            uuid.UUID    `json:"id"`
	Alias         string       `json:"alias"`
	Name          string       `json:"name"`
	Category      string       `json:"category"`
	ImageURL      *string      `json:"image_url"`
	Description   string       `json:"description"`
	SpreadDetails *string      `json:"spread_details"`
	MatchWeather  StringArray  `json:"match_weather"`
	Symptoms      InfoSections `json:"symptoms"`
	Prevention    InfoSections `json:"prevention"`
	Treatment     InfoSections `json:"treatment"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
