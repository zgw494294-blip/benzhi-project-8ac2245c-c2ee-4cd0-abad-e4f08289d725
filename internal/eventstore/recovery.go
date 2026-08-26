package eventstore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"subsurface-survey-gate/internal/domain"
)

func (s *Store) recover() error {
	path := filepath.Join(s.dir, "events.jsonl")
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReaderSize(f, 1024*1024)
	lineNo := 0
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNo++
			var r ledgerRecord
			if err := json.Unmarshal(line, &r); err != nil {
				return fmt.Errorf("账本第 %d 行无效: %w", lineNo, err)
			}
			if r.SchemaVersion != schemaVersion || r.Sequence != s.sequences[r.CampaignID]+1 || r.PreviousDigest != s.roots[r.CampaignID] || recordDigest(r) != r.Digest {
				return fmt.Errorf("账本第 %d 行摘要链校验失败", lineNo)
			}
			if r.State == nil || r.State.ID != r.CampaignID {
				return fmt.Errorf("账本第 %d 行投影无效", lineNo)
			}
			s.campaigns[r.CampaignID], s.roots[r.CampaignID], s.sequences[r.CampaignID] = domain.Clone(r.State), r.Digest, r.Sequence
			s.audit[r.CampaignID] = append(s.audit[r.CampaignID], r)
			if r.Idempotency != nil {
				s.idempotency[r.Idempotency.CampaignID+"|"+r.Idempotency.Key] = *r.Idempotency
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}
