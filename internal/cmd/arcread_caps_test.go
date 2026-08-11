package cmd

import (
	"testing"

	"github.com/hrodrig/groot/internal/arcread"
)

func TestArchiveCapsFromMaxDecompressed(t *testing.T) {
	t.Parallel()
	def := archiveCapsFromMaxDecompressed(0)
	if def.MaxDecompressedBytes != arcread.DefaultMaxDecompressedBytes {
		t.Fatalf("0 override: MaxDecompressedBytes=%d want default %d",
			def.MaxDecompressedBytes, arcread.DefaultMaxDecompressedBytes)
	}
	if def.MaxMemberBytes != arcread.DefaultMaxMemberBytes {
		t.Fatalf("member cap mutated: %d", def.MaxMemberBytes)
	}

	custom := archiveCapsFromMaxDecompressed(100)
	if custom.MaxDecompressedBytes != 100 {
		t.Fatalf("custom override: got %d want 100", custom.MaxDecompressedBytes)
	}
}
