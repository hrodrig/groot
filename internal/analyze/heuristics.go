package analyze

// runHeuristics dispatches evidence scanners. Plan 02-01 implements CrashLoop only;
// remaining kinds land in Plan 02-02.
func runHeuristics(ev evidence) []Hint {
	var hints []Hint
	if h, ok := detectCrashLoop(ev); ok {
		hints = append(hints, h)
	}
	return hints
}
