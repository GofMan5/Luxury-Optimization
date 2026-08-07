package optimizer

import (
	"context"
	"os"
	"testing"
)

func TestWindowsBackgroundSnapshotReadsCurrentProcessIdentity(t *testing.T) {
	snapshot, err := readBackgroundSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pid := uint32(os.Getpid())
	counter, found := snapshot.Counters[pid]
	if !found || counter.Created == 0 || counter.Name == "" {
		t.Fatalf("current process missing from snapshot: %+v", counter)
	}
	if path := backgroundExecutablePath(pid, counter.Created); path == "" {
		t.Fatal("current process executable identity was not resolved")
	}
}
