package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// BoardConf holds board configuration for a project.
type BoardConf struct {
	ID   int    `mapstructure:"id"`
	Name string `mapstructure:"name"`
	Type string `mapstructure:"type"`
}

// ProjectEntry holds per-project configuration.
type ProjectEntry struct {
	Key   string     `mapstructure:"key"`
	Name  string     `mapstructure:"name"`
	Type  string     `mapstructure:"type"`
	Board *BoardConf `mapstructure:"board"`
}

// FindProjectEntry looks up a project entry by map key, project key, or display name.
func FindProjectEntry(projects map[string]ProjectEntry, identifier string) (ProjectEntry, bool) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || len(projects) == 0 {
		return ProjectEntry{}, false
	}

	if entry, ok := projects[identifier]; ok {
		return entry, true
	}

	needle := strings.ToLower(identifier)
	for mapKey, entry := range projects {
		if strings.ToLower(mapKey) == needle {
			return entry, true
		}
		if strings.EqualFold(entry.Key, identifier) {
			return entry, true
		}
		if entry.Name != "" && strings.EqualFold(entry.Name, identifier) {
			return entry, true
		}
	}

	return ProjectEntry{}, false
}

// ApplyProjectContext overlays per-project settings from the projects map onto viper.
// When the identifier is not found in projects, top-level project and board settings are kept.
func ApplyProjectContext(identifier string) {
	var projects map[string]ProjectEntry
	if err := viper.UnmarshalKey("projects", &projects); err != nil || len(projects) == 0 {
		return
	}

	entry, ok := FindProjectEntry(projects, identifier)
	if !ok {
		return
	}

	if entry.Key != "" {
		viper.Set("project.key", entry.Key)
	}
	if entry.Type != "" {
		viper.Set("project.type", entry.Type)
	}

	if entry.Board != nil && entry.Board.ID != 0 {
		viper.Set("board.id", entry.Board.ID)
		viper.Set("board.name", entry.Board.Name)
		viper.Set("board.type", entry.Board.Type)
		return
	}

	viper.Set("board.id", 0)
	viper.Set("board.name", "")
	viper.Set("board.type", "")
}

// ProjectEntryFromConf builds a project entry for the projects map.
func ProjectEntryFromConf(project *projectConf, board *BoardConf) map[string]any {
	if project == nil {
		return nil
	}

	entry := map[string]any{
		"key":  project.Key,
		"type": project.Type,
	}
	if board != nil && board.ID != 0 {
		entry["board"] = board
	}
	return entry
}

// ExistingProjectsMap returns the configured projects map.
func ExistingProjectsMap() map[string]any {
	projects := make(map[string]any)
	var existing map[string]any
	if err := viper.UnmarshalKey("projects", &existing); err == nil {
		for key, entry := range existing {
			projects[key] = entry
		}
	}
	return projects
}

// SaveProjectEntry merges a project entry into the config file.
func SaveProjectEntry(project *projectConf, board *BoardConf, displayName string) error {
	if project == nil {
		return fmt.Errorf("project not selected")
	}

	projects := ExistingProjectsMap()
	entry := ProjectEntryFromConf(project, board)
	if displayName != "" {
		entry["name"] = displayName
	}
	projects[project.Key] = entry

	viper.Set("projects", projects)
	return viper.WriteConfig()
}
