package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dvgamerr/claude-status/internal/model"
)

const (
	appDirectory = "claude-status"
	latestFile   = "latest.json"
)

type Store struct {
	dir string
}

func DefaultDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CLAUDE_STATUS_STATE_DIR")); override != "" {
		return filepath.Clean(override), nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(cacheDir, appDirectory), nil
}

func New(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("state directory is empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve state directory: %w", err)
	}
	return &Store{dir: filepath.Clean(abs)}, nil
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) Save(snapshot model.Snapshot) error {
	if snapshot.SchemaVersion != model.CurrentSchemaVersion {
		return fmt.Errorf("unsupported snapshot schema version %d", snapshot.SchemaVersion)
	}
	if strings.TrimSpace(snapshot.Session.ID) == "" {
		return errors.New("snapshot session ID is empty")
	}

	sessionsDir := filepath.Join(s.dir, "sessions")
	if err := ensurePrivateDir(s.dir); err != nil {
		return err
	}
	if err := ensurePrivateDir(sessionsDir); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	data = append(data, '\n')

	if err := writeAtomic(sessionsDir, sessionFilename(snapshot.Session.ID), data); err != nil {
		return fmt.Errorf("write session snapshot: %w", err)
	}
	if err := writeAtomic(s.dir, latestFile, data); err != nil {
		return fmt.Errorf("write latest snapshot: %w", err)
	}
	return nil
}

func (s *Store) LoadLatest() (model.Snapshot, error) {
	return readSnapshot(filepath.Join(s.dir, latestFile))
}

func (s *Store) LoadAll() ([]model.Snapshot, error) {
	dir := filepath.Join(s.dir, "sessions")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	snapshots := make([]model.Snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		snapshot, err := readSnapshot(filepath.Join(dir, entry.Name()))
		if err != nil {
			// One interrupted or manually edited file must not take down the TUI.
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].CapturedAt.After(snapshots[j].CapturedAt)
	})
	return snapshots, nil
}

func ensurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create state directory %q: %w", path, err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("secure state directory %q: %w", path, err)
	}
	return nil
}

func writeAtomic(dir, name string, data []byte) (returnErr error) {
	temp, err := os.CreateTemp(dir, ".snapshot-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := temp.Name()
	defer func() {
		_ = temp.Close()
		if returnErr != nil {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("secure temporary file: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}

	destination := filepath.Join(dir, name)
	if err := os.Rename(tempName, destination); err != nil {
		return fmt.Errorf("replace %q: %w", destination, err)
	}
	if err := os.Chmod(destination, 0o600); err != nil {
		return fmt.Errorf("secure %q: %w", destination, err)
	}
	return nil
}

func readSnapshot(path string) (model.Snapshot, error) {
	var snapshot model.Snapshot
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot, err
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode %q: %w", path, err)
	}
	if snapshot.SchemaVersion != model.CurrentSchemaVersion {
		return snapshot, fmt.Errorf("unsupported snapshot schema version %d", snapshot.SchemaVersion)
	}
	if snapshot.Session.ID == "" {
		return snapshot, errors.New("snapshot has no session ID")
	}
	return snapshot, nil
}

func sessionFilename(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:12]) + ".json"
}
