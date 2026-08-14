package cmd

import (
	"errors"
	"fmt"
)

// Exit code taxonomy (0.9.x #82; SPEC §3 — exit semantics).
//
//	0  success
//	1  config validation failure (YAML, --since, kubeconfig path, on-disk check)
//	2  Kubernetes client / API error (auth, list, handshake)
//	3  collect aborted (timeout, archive failure, mandatory job failed)
//	4  notify delivery failed (after a successful collect)
//	5  partial job failures ≥ configured threshold, triggered by `--strict` flag
//	   (default threshold 1, configurable via `--strict-threshold`; see plan-0.9.0)
//
// The contract is: any zero return from `groot` means the archive exists on
// disk and notify succeeded. Non-zero tells scripts which subsystem failed
// without forcing them to scrape stderr.
const (
	ExitSuccess         = 0
	ExitConfigError     = 1
	ExitKubernetesError = 2
	ExitCollectAborted  = 3
	ExitNotifyFailed    = 4
	// ExitPartialFailed (5) — triggered by `--strict` flag when partial
	// failures >= strictThreshold (default 1). Wired in pkg/cmd/root.go.
	ExitPartialFailed = 5
)

// exitCoder is the boundary every Cobra RunE funnels errors through. Returning
// `*ExitError` from `RunE` is the only way to communicate a non-default exit
// code through the existing `cmd.Execute() → os.Exit(exitCode(err))` plumbing.
type exitCoder interface {
	error
	ExitCode() int
}

// ExitError wraps a Cobra RunE error with a stable exit code. Always wrap with
// fmt.Errorf("…: %w", &ExitError{...}) — the %w keeps the original cause in
// the error chain so callers can `errors.As` it without losing context.
type ExitError struct {
	Code int
	Err  error
}

// NewExitError builds a wrapped ExitError. Cause may be nil (still useful when
// wrapping a sentinel — e.g. config.ErrConfigValidation).
func NewExitError(code int, cause error) *ExitError {
	return &ExitError{Code: code, Err: cause}
}

// NewExitErrorf builds a wrapped ExitError from a formatted message.
func NewExitErrorf(code int, format string, a ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, a...)}
}

// Error implements the error interface and unwraps the underlying cause.
func (e *ExitError) Error() string {
	if e == nil {
		return "<nil exit error>"
	}
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return fmt.Sprintf("exit code %d: %s", e.Code, e.Err.Error())
}

// Unwrap exposes the wrapped error so errors.Is / errors.As work normally.
func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode satisfies exitCoder — main.exitCode reads this.
func (e *ExitError) ExitCode() int {
	if e == nil {
		return ExitSuccess
	}
	if e.Code < 0 || e.Code > 255 {
		// os.Exit masks to 8 bits; clamp to keep behavior predictable and
		// avoid stderr/logs that lie about the configured code.
		return ExitSuccess
	}
	return e.Code
}

// ExitCodeOf returns the exit code associated with err, defaulting to 1
// for plain errors and 0 for nil. It walks the error chain via errors.As so
// callers wrapping with %w still get the intended code.
func ExitCodeOf(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var ec exitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return ExitConfigError
}
