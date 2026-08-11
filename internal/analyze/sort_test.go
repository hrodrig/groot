package analyze

import "testing"

func TestSortHints_SeverityThenKind(t *testing.T) {
	hints := []Hint{
		{Kind: KindNotReady, Severity: SeverityWarn, Evidence: []Evidence{{Path: "b"}}},
		{Kind: KindEvicted, Severity: SeverityError, Evidence: []Evidence{{Path: "a"}}},
		{Kind: KindCrashLoopBackOff, Severity: SeverityError, Evidence: []Evidence{{Path: "c"}}},
		{Kind: KindImagePullBackOff, Severity: SeverityInfo, Evidence: []Evidence{{Path: "d"}}},
	}
	sortHints(hints)
	want := []Kind{KindCrashLoopBackOff, KindEvicted, KindNotReady, KindImagePullBackOff}
	if len(hints) != len(want) {
		t.Fatalf("len=%d", len(hints))
	}
	for i, k := range want {
		if hints[i].Kind != k {
			t.Fatalf("hints[%d]=%s want %s (full=%v)", i, hints[i].Kind, k, kindsOf(hints))
		}
	}
	// Within same severity, Kind ascending: CrashLoopBackOff < Evicted
	if hints[0].Severity != SeverityError || hints[1].Severity != SeverityError {
		t.Fatalf("first two should be error")
	}
	if hints[2].Severity != SeverityWarn {
		t.Fatalf("third should be warn")
	}
	if hints[3].Severity != SeverityInfo {
		t.Fatalf("fourth should be info")
	}
}

func kindsOf(hints []Hint) []Kind {
	out := make([]Kind, len(hints))
	for i, h := range hints {
		out[i] = h.Kind
	}
	return out
}
