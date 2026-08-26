package eventstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func (s *Store) writeSnapshot(id string, seq int64, root string, state *domain.SurveyCampaign) error {
	dir := filepath.Join(s.dir, "snapshots")
	tmp, err := os.CreateTemp(dir, ".snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	snap := snapshotFile{SchemaVersion: schemaVersion, CampaignID: id, Sequence: seq, ChainRoot: root, State: domain.Clone(state), StateDigest: domain.Digest(state), CreatedAt: time.Now().UTC()}
	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	if err = enc.Encode(snap); err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, filepath.Join(dir, id+".json")); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	closeErr = d.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	ok = true
	return nil
}
