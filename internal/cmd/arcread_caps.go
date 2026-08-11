package cmd

import "github.com/hrodrig/groot/internal/arcread"

// archiveCapsFromMaxDecompressed returns DefaultCaps, optionally overriding
// MaxDecompressedBytes when maxDecompressed > 0 (CLI --max-decompressed).
func archiveCapsFromMaxDecompressed(maxDecompressed int64) arcread.Caps {
	caps := arcread.DefaultCaps()
	if maxDecompressed > 0 {
		caps.MaxDecompressedBytes = maxDecompressed
	}
	return caps
}
