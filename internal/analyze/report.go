package analyze

import (
	"encoding/json"
	"fmt"
)

// Severity ranks hint urgency for deterministic sorting (error > warn > info).
type Severity int

const (
	SeverityError Severity = iota // highest priority in sort
	SeverityWarn
	SeverityInfo
)

// Kind values — verbatim heuristic set (D-04).
type Kind string

const (
	KindCrashLoopBackOff Kind = "CrashLoopBackOff"
	KindOOMKilled        Kind = "OOMKilled"
	KindImagePullBackOff Kind = "ImagePullBackOff"
	KindNotReady         Kind = "NotReady"
	KindEvicted          Kind = "Evicted"
)

// Evidence cites an archive member path with an optional bounded excerpt.
type Evidence struct {
	Path    string `json:"path"`
	Excerpt string `json:"excerpt,omitempty"`
}

// Hint is one evidence-backed hypothesis (not a definitive diagnosis).
type Hint struct {
	Kind          Kind       `json:"kind"`
	Severity      Severity   `json:"severity"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Evidence      []Evidence `json:"evidence"`
	OpenQuestions []string   `json:"open_questions,omitempty"`
}

// Note records non-fatal degrade information (missing members, partial parse).
type Note struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// Report is the format-agnostic analyze result (Markdown/JSON/LLM renderers).
type Report struct {
	RunID                string `json:"run_id,omitempty"`
	ArchiveSHA256        string `json:"archive_sha256,omitempty"`
	ArchiveLayoutVersion int    `json:"archive_layout_version,omitempty"`
	ArchivePath          string `json:"archive_path,omitempty"`
	Hints                []Hint `json:"hints"`
	Notes                []Note `json:"notes,omitempty"`
	Summary              string `json:"summary"`
}

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarn:
		return "warn"
	case SeverityInfo:
		return "info"
	default:
		return "info"
	}
}

// MarshalJSON encodes Severity as "error"|"warn"|"info".
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes Severity from "error"|"warn"|"info".
func (s *Severity) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v {
	case "error":
		*s = SeverityError
	case "warn":
		*s = SeverityWarn
	case "info":
		*s = SeverityInfo
	default:
		return fmt.Errorf("unknown severity %q", v)
	}
	return nil
}

// Rank returns sort priority (lower = more severe).
func (s Severity) Rank() int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarn:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}
