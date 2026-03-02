package models

// Membership is the pivot table linking users to teams with a role.
type Membership struct {
	Base
	UserID uint   `json:"user_id"`
	TeamID string `gorm:"type:uuid" json:"team_id"`
	Role   string `gorm:"default:member" json:"role"` // owner | admin | member | viewer
	User   User   `gorm:"foreignKey:UserID" json:"-"`
	Team   Team   `gorm:"foreignKey:TeamID" json:"-"`
}
