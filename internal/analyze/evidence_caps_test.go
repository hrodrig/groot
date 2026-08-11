package analyze

import "testing"

func TestAnalyzeLocalCaps(t *testing.T) {
	t.Parallel()
	if capEventsBytes != 2<<20 {
		t.Fatalf("capEventsBytes=%d want 2 MiB", capEventsBytes)
	}
	if capTextBytes != 32<<20 {
		t.Fatalf("capTextBytes=%d want 32 MiB", capTextBytes)
	}
	// Stay under arcread DefaultMaxMemberBytes (64 MiB) so Open can index the member.
	if capTextBytes >= 64<<20 {
		t.Fatalf("capTextBytes must stay below arcread 64 MiB member cap")
	}
}
