package handlers

import (
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// userTeamIDs returns the team IDs the user belongs to via memberships.
func userTeamIDs(db *gorm.DB, userID uint) []string {
	var ids []string
	db.Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Pluck("team_id", &ids)
	return ids
}

// teamScopedProjects returns a query scoping projects to the user's own
// projects OR projects belonging to any of the user's teams.
func teamScopedProjects(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}

// teamScopedFeatures returns a query scoping features to the user's own
// features OR features belonging to any of the user's teams.
func teamScopedFeatures(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}

// teamScopedRequests returns a query scoping requests to the user's own
// requests OR requests belonging to any of the user's teams.
func teamScopedRequests(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}

// teamScopedPersons returns a query scoping persons to the user's own
// persons OR persons belonging to any of the user's teams.
func teamScopedPersons(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}
