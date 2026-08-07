package optimizer

import (
	"context"
	"os"
	"testing"
)

func TestLinuxBackgroundSnapshotReadsCurrentProcessIdentity(t *testing.T) {
	if !backgroundAdvisorAvailable() {
		t.Skip("proc process counters unavailable")
	}
	snapshot, err := readBackgroundSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pid := uint32(os.Getpid())
	counter, found := snapshot.Counters[pid]
	if !found || counter.Created == 0 || counter.Name == "" || snapshot.TotalCPU == 0 {
		t.Fatalf("current process missing from snapshot: %+v", counter)
	}
	if path := backgroundExecutablePath(pid, counter.Created); path == "" {
		t.Fatal("current process executable identity was not resolved")
	}
}
