package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFindProjectEntry(t *testing.T) {
	projects := map[string]ProjectEntry{
		"PROJA": {
			Key:  "PROJA",
			Name: "Project Alpha",
			Board: &BoardConf{
				ID:   100,
				Name: "Alpha Board",
				Type: "kanban",
			},
		},
		"PROJB": {
			Key:  "PROJB",
			Name: "Project Beta",
		},
	}

	entry, ok := FindProjectEntry(projects, "PROJB")
	assert.True(t, ok)
	assert.Equal(t, "PROJB", entry.Key)

	entry, ok = FindProjectEntry(projects, "Project Beta")
	assert.True(t, ok)
	assert.Equal(t, "PROJB", entry.Key)

	_, ok = FindProjectEntry(projects, "UNKNOWN")
	assert.False(t, ok)
}

func TestApplyProjectContext(t *testing.T) {
	viper.Reset()
	viper.Set("project.key", "PROJA")
	viper.Set("board.id", 100)
	viper.Set("board.name", "Alpha Board")
	viper.Set("projects", map[string]any{
		"PROJA": map[string]any{
			"key":  "PROJA",
			"type": "",
			"board": map[string]any{
				"id":   100,
				"name": "Alpha Board",
				"type": "kanban",
			},
		},
		"PROJB": map[string]any{
			"key":  "PROJB",
			"name": "Project Beta",
			"type": "",
		},
	})

	ApplyProjectContext("PROJB")

	assert.Equal(t, "PROJB", viper.GetString("project.key"))
	assert.Equal(t, 0, viper.GetInt("board.id"))
	assert.Equal(t, "", viper.GetString("board.name"))
}

func TestSaveProjectEntry(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := dir + "/.config.yml"
	viper.SetConfigFile(cfgFile)
	viper.SetConfigType(FileType)
	viper.Set("installation", "Local")
	viper.Set("server", "https://example.com")
	assert.NoError(t, viper.WriteConfig())

	project := &projectConf{Key: "PROJB", Type: "classic"}
	board := &BoardConf{ID: 42, Name: "Beta Board", Type: "kanban"}

	assert.NoError(t, SaveProjectEntry(project, board, "Project Beta"))

	viper.Reset()
	viper.SetConfigFile(cfgFile)
	viper.SetConfigType(FileType)
	assert.NoError(t, viper.ReadInConfig())

	var projects map[string]ProjectEntry
	assert.NoError(t, viper.UnmarshalKey("projects", &projects))
	assert.Len(t, projects, 1)

	entry, ok := FindProjectEntry(projects, "PROJB")
	assert.True(t, ok)
	assert.Equal(t, "PROJB", entry.Key)
	assert.Equal(t, "Project Beta", entry.Name)
	assert.NotNil(t, entry.Board)
	assert.Equal(t, 42, entry.Board.ID)
}
