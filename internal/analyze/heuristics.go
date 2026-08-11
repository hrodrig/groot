package analyze

// runHeuristics dispatches the five evidence scanners (D-04).
func runHeuristics(ev evidence, notes *[]Note) []Hint {
	var hints []Hint
	if h, ok := detectCrashLoop(ev); ok {
		hints = append(hints, h)
	}
	if h, ok := detectOOMKilled(ev, notes); ok {
		hints = append(hints, h)
	}
	if h, ok := detectImagePullBackOff(ev); ok {
		hints = append(hints, h)
	}
	if h, ok := detectNotReady(ev); ok {
		hints = append(hints, h)
	}
	if h, ok := detectEvicted(ev); ok {
		hints = append(hints, h)
	}
	return hints
}
