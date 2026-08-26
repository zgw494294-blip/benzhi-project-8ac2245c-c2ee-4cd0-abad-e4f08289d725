package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Store) writeProjection() error {
	temp, err := os.CreateTemp(s.dir, ".projection-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.state); err != nil {
		cleanup()
		return err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temp.Chmod(0o640); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, s.projectionPath); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	dir, err := os.Open(filepath.Dir(s.projectionPath))
	if err != nil {
		return fmt.Errorf("打开投影目录: %w", err)
	}
	defer dir.Close()
	return dir.Sync()
}
