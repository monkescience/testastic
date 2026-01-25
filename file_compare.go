package testastic

// compareFileLines compares expected and actual lines.
func compareFileLines(expected, actual []string) []Difference {
	var diffs []Difference

	maxLines := max(len(expected), len(actual))

	for i := range maxLines {
		var expLine, actLine string
		var hasExp, hasAct bool

		if i < len(expected) {
			expLine = expected[i]
			hasExp = true
		}
		if i < len(actual) {
			actLine = actual[i]
			hasAct = true
		}

		if !hasExp {
			// Extra line in actual
			diffs = append(diffs, Difference{
				Path:     lineNumberPath(i + 1),
				Expected: nil,
				Actual:   actLine,
				Type:     DiffAdded,
			})
			continue
		}

		if !hasAct {
			// Missing line in actual
			diffs = append(diffs, Difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine,
				Actual:   nil,
				Type:     DiffRemoved,
			})
			continue
		}

		// Both lines exist - compare them
		if expLine != actLine {
			diffs = append(diffs, Difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine,
				Actual:   actLine,
				Type:     DiffChanged,
			})
		}
	}

	return diffs
}

// lineNumberPath returns a path string for a line number.
func lineNumberPath(lineNum int) string {
	return "line " + itoa(lineNum)
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
