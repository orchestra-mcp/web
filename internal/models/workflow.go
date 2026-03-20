package models

import "gorm.io/datatypes"

// Workflow represents a workflow definition synced from MCP (DB-backed, team-sharable).
// Maps to the MCP globaldb.WorkflowRecord schema with user/team ownership.
type Workflow struct {
	Base
	UserID       uint           `gorm:"uniqueIndex:idx_wf_user_wid" json:"user_id"`
	TeamID       *string        `json:"team_id"`
	ProjectSlug  string         `gorm:"index" json:"project_slug"`
	WorkflowID   string         `gorm:"uniqueIndex:idx_wf_user_wid" json:"workflow_id"` // e.g. WFL-XXX
	Name         string         `json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	InitialState string         `gorm:"default:todo" json:"initial_state"`
	IsDefault    bool           `gorm:"default:true" json:"is_default"`
	States       datatypes.JSON `json:"states"`      // map[string]{label,terminal,active_work}
	Transitions  datatypes.JSON `json:"transitions"` // [{from,to,gate}]
	Gates        datatypes.JSON `json:"gates"`       // map[string]{label,required_section,file_patterns,docs_folder,skippable_for}
	Version      int            `gorm:"default:1" json:"version"`
	Meta         datatypes.JSON `json:"meta"`
}
