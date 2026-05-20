package lib

import (
	"encoding/json"
	"fmt"
	"time"
)

type Entry struct {
	Timestamp time.Time
	Name      string
	Labels    []string
}

func (e Entry) IsBreak() bool {
	return len(e.Name) == 0
}

func TimestampToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0).UTC()
}

func (e Entry) Time() time.Time {
	return e.Timestamp
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	var tmp []json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if len(tmp) < 3 {
		return fmt.Errorf("invalid entry format: expected at least 3 elements, got %d", len(tmp))
	}

	if err := json.Unmarshal(tmp[0], &e.Timestamp); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &e.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[2], &e.Labels); err != nil {
		return err
	}

	return nil
}

func (e Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		e.Timestamp,
		e.Name,
		e.Labels,
	})
}
