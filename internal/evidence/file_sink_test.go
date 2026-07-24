package evidence

import (
	"bufio"
	"encoding/json"
	"github.com/jhansi-io/jhansi/internal/domain"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSinkRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	sink, err := NewFileSink(path)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}

	at := time.Now()
	in := []domain.Event{
		{Name: "sandbox.created", At: at, AggregateID: "sb_1"},
		{Name: "sandbox_deleted", At: at, AggregateID: "sb_1"},
	}
	if err := sink.Record(in); err != nil {
		t.Fatalf("Record: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var got []domain.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e domain.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("unmarshal %q: %v", scanner.Text(), err)
		}
		got = append(got, e)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2", len(got))
	}
	if got[0].Name != "sandbox.created" || got[1].Name != "sandbox_deleted" {
		t.Fatalf("wrong names/order: %q, %q", got[0].Name, got[1].Name)
	}
	if got[0].AggregateID != "sb_1" {
		t.Fatalf("aggregate id: %q", got[0].AggregateID)
	}

}
