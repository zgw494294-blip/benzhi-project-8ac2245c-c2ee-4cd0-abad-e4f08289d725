package ledger

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func (s *Store) recover() error {
	fromEvents, err := s.rebuildFromEvents()
	if err != nil {
		return err
	}
	loaded, loadErr := s.loadProjection()
	if loadErr == nil && loaded.LastSequence == fromEvents.LastSequence && loaded.LastDigest == fromEvents.LastDigest {
		s.state = loaded
		return nil
	}
	s.state = fromEvents
	if err := s.writeProjection(); err != nil {
		return fmt.Errorf("重建投影: %w", err)
	}
	return nil
}

func (s *Store) rebuildFromEvents() (projection, error) {
	state := emptyProjection()
	f, err := os.Open(s.eventsPath)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("打开事件账本: %w", err)
	}
	defer f.Close()
	decoder := json.NewDecoder(bufio.NewReader(f))
	for {
		var event Event
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return state, fmt.Errorf("解析事件 %d: %w", state.LastSequence+1, err)
		}
		if err := validateEvent(event, state.LastSequence, state.LastDigest); err != nil {
			return state, err
		}
		s.state = state
		s.applyEvent(event)
		state = s.state
	}
	return state, nil
}

func validateEvent(event Event, lastSequence int64, lastDigest string) error {
	if event.SchemaVersion != schemaVersion {
		return fmt.Errorf("事件 %d 的 schemaVersion=%d 不受支持", event.Sequence, event.SchemaVersion)
	}
	if event.Sequence != lastSequence+1 {
		return fmt.Errorf("事件序号断裂: 期望 %d，实际 %d", lastSequence+1, event.Sequence)
	}
	if event.PreviousDigest != lastDigest {
		return fmt.Errorf("事件 %d 的前序摘要不匹配", event.Sequence)
	}
	if event.EventType == "" {
		return fmt.Errorf("事件 %d 缺少 eventType", event.Sequence)
	}
	if event.Campaign != nil && (event.Campaign.ID != event.CampaignID || event.Campaign.Version != event.CampaignVersion) {
		return fmt.Errorf("事件 %d 的周期标识或版本与载荷不一致", event.Sequence)
	}
	if event.Campaign == nil && event.Credential == nil {
		return fmt.Errorf("事件 %d 不包含可投影载荷", event.Sequence)
	}
	if event.Credential != nil && event.Credential.CampaignID != event.CampaignID {
		return fmt.Errorf("事件 %d 的凭据周期不一致", event.Sequence)
	}
	checksum, err := event.calculateChecksum()
	if err != nil {
		return err
	}
	if checksum != event.Checksum {
		return fmt.Errorf("事件 %d 校验和不匹配", event.Sequence)
	}
	return nil
}

func (s *Store) loadProjection() (projection, error) {
	state := emptyProjection()
	f, err := os.Open(s.projectionPath)
	if err != nil {
		return state, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return state, err
	}
	if state.SchemaVersion != schemaVersion || state.Campaigns == nil || state.Idempotency == nil || state.Credentials == nil {
		return state, fmt.Errorf("投影模式无效")
	}
	return state, nil
}
