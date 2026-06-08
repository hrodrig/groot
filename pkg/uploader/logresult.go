package uploader

import (
	"fmt"
	"strings"
)

// FormatResult returns a safe log line without credential metadata.
func FormatResult(r *Result) string {
	if r == nil {
		return ""
	}
	etag := strings.TrimSpace(r.ETag)
	if etag != "" {
		return fmt.Sprintf("%s uploaded %s (size=%d etag=%s)", r.Provider, r.URI, r.Size, etag)
	}
	return fmt.Sprintf("%s uploaded %s (size=%d)", r.Provider, r.URI, r.Size)
}
