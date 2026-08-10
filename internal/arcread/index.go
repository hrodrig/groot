package arcread

import "strings"

// MemberMeta describes one indexed regular-file member from Pass 1.
type MemberMeta struct {
	Name     string
	Size     int64
	Typeflag byte
	Ordinal  int // stream order among indexed entries; used by ReadMember
}

// Members returns regular-file members in archive stream order.
func (a *Archive) Members() []MemberMeta {
	if a == nil {
		return nil
	}
	out := make([]MemberMeta, len(a.members))
	copy(out, a.members)
	return out
}

// Lookup returns the member with an exact path match.
func (a *Archive) Lookup(name string) (MemberMeta, bool) {
	if a == nil {
		return MemberMeta{}, false
	}
	for _, m := range a.members {
		if m.Name == name {
			return m, true
		}
	}
	return MemberMeta{}, false
}

// LookupSuffix finds the first member whose name equals suffix or ends with "/"+suffix.
func (a *Archive) LookupSuffix(suffix string) (MemberMeta, bool) {
	if a == nil || suffix == "" {
		return MemberMeta{}, false
	}
	if m, ok := a.Lookup(suffix); ok {
		return m, true
	}
	want := "/" + strings.TrimPrefix(suffix, "/")
	for _, m := range a.members {
		if strings.HasSuffix(m.Name, want) {
			return m, true
		}
	}
	return MemberMeta{}, false
}
