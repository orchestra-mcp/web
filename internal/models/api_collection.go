package models

import "gorm.io/datatypes"

// ApiCollection groups related API endpoints (like a Postman collection).
type ApiCollection struct {
	Base
	UserID      uint           `gorm:"index" json:"user_id"`
	TeamID      *string        `gorm:"type:uuid;index" json:"team_id,omitempty"`
	Name        string         `json:"name"`
	Slug        string         `gorm:"index" json:"slug"`
	Description string         `json:"description"`
	BaseURL     string         `json:"base_url"`
	AuthType    string         `gorm:"default:none" json:"auth_type"` // none, bearer, basic, api_key, oauth2
	AuthConfig  datatypes.JSON `json:"auth_config"`
	Variables   datatypes.JSON `json:"variables"` // collection-level variables
	Visibility  string         `gorm:"default:private" json:"visibility"` // private, team, public
	Version     int            `gorm:"default:1" json:"version"`
}

// ApiEndpoint represents a single API request within a collection.
type ApiEndpoint struct {
	Base
	CollectionID string         `gorm:"type:uuid;index" json:"collection_id"`
	UserID       uint           `gorm:"index" json:"user_id"`
	Name         string         `json:"name"`
	Method       string         `gorm:"default:GET" json:"method"` // GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
	Path         string         `json:"path"`
	Headers      datatypes.JSON `json:"headers"`
	QueryParams  datatypes.JSON `json:"query_params"`
	Body         string         `gorm:"type:text" json:"body"`
	BodyType     string         `gorm:"default:none" json:"body_type"` // none, json, form, xml, raw
	Description  string         `json:"description"`
	SortOrder    int            `gorm:"default:0" json:"sort_order"`
	FolderPath   string         `json:"folder_path"` // virtual folder within collection
}

// ApiEnvironment stores environment-specific variables for API collections.
type ApiEnvironment struct {
	Base
	CollectionID string         `gorm:"type:uuid;index" json:"collection_id"`
	UserID       uint           `gorm:"index" json:"user_id"`
	Name         string         `json:"name"`
	Variables    datatypes.JSON `json:"variables"` // key-value pairs
	IsActive     bool           `gorm:"default:false" json:"is_active"`
}
