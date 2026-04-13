package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScheduledTaskRunSlotDailyCatchesUpMissedHour(t *testing.T) {
	task := ScheduledTask{
		Name:     "Dream Daily",
		Schedule: "daily",
		Hours:    []int{4},
	}

	now := time.Date(2026, 4, 14, 4, 1, 0, 0, time.FixedZone("CST", 8*3600))
	slot, due, err := task.RunSlot(now)
	if err != nil {
		t.Fatalf("RunSlot returned error: %v", err)
	}
	if !due {
		t.Fatalf("expected task to be due at %s", now.Format(time.RFC3339))
	}
	if slot != "2026-04-14T04" {
		t.Fatalf("expected slot 2026-04-14T04, got %q", slot)
	}
}

func TestScheduledTaskRunSlotDailyBeforeFirstHourIsNotDue(t *testing.T) {
	task := ScheduledTask{
		Name:     "Dream Daily",
		Schedule: "daily",
		Hours:    []int{4},
	}

	now := time.Date(2026, 4, 14, 3, 59, 0, 0, time.FixedZone("CST", 8*3600))
	slot, due, err := task.RunSlot(now)
	if err != nil {
		t.Fatalf("RunSlot returned error: %v", err)
	}
	if due {
		t.Fatalf("expected task to be not due at %s", now.Format(time.RFC3339))
	}
	if slot != "" {
		t.Fatalf("expected empty slot, got %q", slot)
	}
}

func TestSchedulerPersistsDailyCatchUpState(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cron.json")
	tasks := []ScheduledTask{
		{
			Name:     "Dream Daily",
			Schedule: "daily",
			Type:     "ai",
			Prompt:   "read INIT.md and DREAM.md and do",
			Hours:    []int{4},
			Enabled:  boolPtr(true),
		},
	}
	data, err := json.Marshal(tasks)
	if err != nil {
		t.Fatalf("marshal tasks: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write cron config: %v", err)
	}

	now := time.Date(2026, 4, 14, 4, 30, 0, 0, time.FixedZone("CST", 8*3600))
	dispatchCh := make(chan DispatchRequest, 2)

	s1 := NewScheduler(configPath, dispatchCh)
	s1.runDueTasks(now)
	assertDispatchCount(t, dispatchCh, 1)

	s2 := NewScheduler(configPath, dispatchCh)
	if err := s2.loadState(); err != nil {
		t.Fatalf("load state: %v", err)
	}
	s2.runDueTasks(now)
	assertDispatchCount(t, dispatchCh, 0)
}

func assertDispatchCount(t *testing.T, ch <-chan DispatchRequest, want int) {
	t.Helper()
	got := 0
	for {
		select {
		case <-ch:
			got++
		default:
			if got != want {
				t.Fatalf("expected %d dispatches, got %d", want, got)
			}
			return
		}
	}
}

func boolPtr(v bool) *bool {
	return &v
}
