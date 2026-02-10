package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore implements StateStore with one JSON file per team.
// Writes are atomic (tmp + rename). Per-team mutexes prevent concurrent writes.
type FileStore struct {
	dir   string
	locks sync.Map // map[string]*sync.Mutex
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

func (fs *FileStore) teamMutex(teamName string) *sync.Mutex {
	v, _ := fs.locks.LoadOrStore(teamName, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (fs *FileStore) filePath(teamName string) string {
	return filepath.Join(fs.dir, teamName+".json")
}

func (fs *FileStore) Load(_ context.Context, teamName string) (*TeamState, error) {
	mu := fs.teamMutex(teamName)
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(fs.filePath(teamName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state TeamState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	return &state, nil
}

func (fs *FileStore) Save(_ context.Context, state *TeamState) error {
	mu := fs.teamMutex(state.TeamName)
	mu.Lock()
	defer mu.Unlock()

	state.LastUpdated = time.Now().UTC()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.MkdirAll(fs.dir, 0755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	tmpPath := fs.filePath(state.TeamName) + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, fs.filePath(state.TeamName)); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

func (fs *FileStore) Delete(_ context.Context, teamName string) error {
	mu := fs.teamMutex(teamName)
	mu.Lock()
	defer mu.Unlock()

	err := os.Remove(fs.filePath(teamName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting state file: %w", err)
	}
	return nil
}
