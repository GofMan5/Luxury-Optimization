package optimizer

import "testing"

func TestParseMountInfoDecodesPathsAndOptions(t *testing.T) {
	mounts := parseMountInfo("36 25 8:1 / / rw,relatime - ext4 /dev/sda1 rw\n37 25 8:2 / /mnt/My\\040Games ro,nosuid - xfs /dev/sda2 ro")
	if len(mounts) != 2 || mounts[1].path != "/mnt/My Games" || !mounts[1].readOnly || mounts[1].fileSystem != "xfs" {
		t.Fatalf("mounts=%+v", mounts)
	}
}
