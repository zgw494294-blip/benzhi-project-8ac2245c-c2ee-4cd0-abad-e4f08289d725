package eventstore

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"subsurface-survey-gate/internal/domain"
)

type Store struct {
	dir         string
	mu          sync.RWMutex
	campaigns   map[string]*domain.SurveyCampaign
	roots       map[string]string
	sequences   map[string]int64
	idempotency map[string]domain.IdempotencyRecord
	audit       map[string][]ledgerRecord
	lockMu      sync.Mutex
	locks       map[string]*sync.Mutex
	appendMu    sync.Mutex
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, domain.Validation("dataDir", "不能为空")
	}
	if err := os.MkdirAll(filepath.Join(dir, "snapshots"), 0750); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, campaigns: map[string]*domain.SurveyCampaign{}, roots: map[string]string{}, sequences: map[string]int64{}, idempotency: map[string]domain.IdempotencyRecord{}, audit: map[string][]ledgerRecord{}, locks: map[string]*sync.Mutex{}}
	if err := s.recover(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) AuditRecords(ctx context.Context, id string) ([]domain.AuditRecord, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	s.mu.RLock()
	records, exists := s.audit[id]
	root := s.roots[id]
	_, campaignExists := s.campaigns[id]
	previous := ""
	out := make([]domain.AuditRecord, len(records))
	for i, record := range records {
		if record.Sequence != int64(i+1) || record.PreviousDigest != previous || record.Digest == "" || recordDigest(record) != record.Digest {
			s.mu.RUnlock()
			return nil, "", domain.IntegrityError("批次事件序号或摘要链不连续")
		}
		out[i] = domain.AuditRecord{Sequence: record.Sequence, PreviousDigest: record.PreviousDigest, Digest: record.Digest, Event: record.Event}
		out[i].Event.Facts = append([]byte(nil), record.Event.Facts...)
		previous = record.Digest
	}
	s.mu.RUnlock()
	if !campaignExists || !exists {
		return nil, "", domain.NotFound("批次")
	}
	if previous != root {
		return nil, "", domain.IntegrityError("批次事件链根与当前账本根不一致")
	}
	return out, root, nil
}

func (s *Store) campaignLock(id string) *sync.Mutex {
	s.lockMu.Lock()
	defer s.lockMu.Unlock()
	if s.locks[id] == nil {
		s.locks[id] = &sync.Mutex{}
	}
	return s.locks[id]
}

func (s *Store) Load(ctx context.Context, id string) (*domain.SurveyCampaign, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	c := domain.Clone(s.campaigns[id])
	s.mu.RUnlock()
	if c == nil {
		return nil, domain.NotFound("批次")
	}
	return c, nil
}

func (s *Store) LookupIdempotency(ctx context.Context, namespace, key string) (*domain.IdempotencyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	record, ok := s.idempotency[namespace+"|"+key]
	s.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	copy := record
	copy.Response = append([]byte(nil), record.Response...)
	return &copy, nil
}

func (s *Store) ChainRoot(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.RLock()
	root, ok := s.roots[id]
	s.mu.RUnlock()
	if !ok {
		return "", domain.NotFound("批次")
	}
	return root, nil
}

func (s *Store) FindCredential(ctx context.Context, id string) (*domain.ReleaseCredential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.campaigns {
		if c.Credential != nil && c.Credential.ID == id {
			copy := *c.Credential
			return &copy, nil
		}
	}
	return nil, domain.NotFound("凭据")
}
