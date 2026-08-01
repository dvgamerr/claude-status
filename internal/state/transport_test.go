package state

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
)

func TestDecodeSnapshotAcceptsSanitizedSchema(t *testing.T) {
	want := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    time.Unix(123, 0).UTC(),
		Provider:      "codex",
		Session:       model.Session{ID: "thread-123"},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeSnapshot(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("DecodeSnapshot() error = %v", err)
	}
	if got.Provider != want.Provider || got.Session.ID != want.Session.ID || !got.CapturedAt.Equal(want.CapturedAt) {
		t.Fatalf("DecodeSnapshot() = %+v, want %+v", got, want)
	}
}

func TestDecodeSnapshotRejectsRawOrMalformedPayloads(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: " ", want: "empty"},
		{name: "unknown raw field", input: `{"schema_version":1,"prompt":"secret"}`, want: "unknown field"},
		{name: "multiple", input: `{}` + "\n" + `{}`, want: "multiple"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeSnapshot(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeSnapshot() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
