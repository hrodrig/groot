package cmd

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCodeOf_nil(t *testing.T) {
	if got := ExitCodeOf(nil); got != ExitSuccess {
		t.Fatalf("nil err => %d, want %d", got, ExitSuccess)
	}
}

func TestExitCodeOf_plainError(t *testing.T) {
	if got := ExitCodeOf(errors.New("boom")); got != ExitConfigError {
		t.Fatalf("plain err => %d, want %d", got, ExitConfigError)
	}
}

func TestExitCodeOf_exitError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"config", NewExitError(ExitConfigError, errors.New("bad yaml")), ExitConfigError},
		{"kubernetes", NewExitError(ExitKubernetesError, errors.New("client")), ExitKubernetesError},
		{"collect-aborted", NewExitError(ExitCollectAborted, errors.New("archive")), ExitCollectAborted},
		{"notify-failed", NewExitError(ExitNotifyFailed, errors.New("slack 500")), ExitNotifyFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeOf(tc.err); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestExitCodeOf_wrappedExitError(t *testing.T) {
	base := NewExitError(ExitKubernetesError, errors.New("list namespaces"))
	wrapped := fmt.Errorf("collect logs: %w", base)
	if got := ExitCodeOf(wrapped); got != ExitKubernetesError {
		t.Fatalf("wrapped exit error must surface its code; got %d", got)
	}
}

func TestExitCodeOf_wrappedPlain(t *testing.T) {
	inner := errors.New("no kubeconfig")
	wrapped := fmt.Errorf("load config: %w", inner)
	if got := ExitCodeOf(wrapped); got != ExitConfigError {
		t.Fatalf("plain wrapped error must default to config code; got %d", got)
	}
}

func TestNewExitErrorf_preservesCode(t *testing.T) {
	err := NewExitErrorf(ExitNotifyFailed, "send slack: %d", 500)
	if got := ExitCodeOf(err); got != ExitNotifyFailed {
		t.Fatalf("got %d want %d", got, ExitNotifyFailed)
	}
	if err.Error() == "" {
		t.Fatal("error message should not be empty")
	}
}

func TestExitError_messages(t *testing.T) {
	if got := (*ExitError)(nil).Error(); got == "" {
		t.Fatal("nil exit error must still produce a message")
	}
	withCause := NewExitError(ExitCollectAborted, errors.New("context deadline"))
	if want := "exit code 3"; !contains(withCause.Error(), want) {
		t.Fatalf("missing code in message: %q", withCause.Error())
	}
	if want := "context deadline"; !contains(withCause.Error(), want) {
		t.Fatalf("missing cause in message: %q", withCause.Error())
	}

	noCause := NewExitError(ExitConfigError, nil)
	if got := noCause.Error(); got == "" {
		t.Fatal("nil-cause error must still produce a message")
	}
}

func TestExitError_clampsOutOfRange(t *testing.T) {
	e := &ExitError{Code: 999, Err: errors.New("oob")}
	if got := e.ExitCode(); got != ExitSuccess {
		t.Fatalf("out-of-range code must clamp to 0; got %d", got)
	}
	e = &ExitError{Code: -3, Err: errors.New("neg")}
	if got := e.ExitCode(); got != ExitSuccess {
		t.Fatalf("negative code must clamp to 0; got %d", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	// Tiny helper to avoid pulling strings into the table test.
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
