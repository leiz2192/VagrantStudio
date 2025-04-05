package internal

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveEnvsWithoutLock(t *testing.T) {
	dpath := t.TempDir()
	tmpfile := filepath.Join(dpath, "test")

	e := &Environment{envs: make(map[int]*Machine)}
	e.envs[0] = &Machine{Name: "test", Path: tmpfile, Stat: "Not Started"}
	err := e.SaveEnvsWithoutLock(tmpfile)
	assert.Nil(t, err)

	clear(e.envs)
	err = e.LoadEnvsWithoutLock(tmpfile)
	assert.Nil(t, err)
	assert.Equal(t, "test", e.envs[0].Name)
	assert.Equal(t, tmpfile, e.envs[0].Path)
	assert.Equal(t, "Not Started", e.envs[0].Stat)
}
