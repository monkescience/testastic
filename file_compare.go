package testastic

// compareFileLines compares expected and actual lines.
func compareFileLines(expected, actual []string) []difference {
	var diffs []difference

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
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: nil,
				Actual:   actLine,
				Type:     diffAdded,
			})

			continue
		}

		if !hasAct {
			// Missing line in actual
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine,
				Actual:   nil,
				Type:     diffRemoved,
			})

			continue
		}

		// Both lines exist - compare them
		if expLine != actLine {
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine,
				Actual:   actLine,
				Type:     diffChanged,
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

// compareFileLinesWithMatchers compares expected and actual lines, supporting matchers.
//
//nolint:funlen // Line-by-line comparison requires sequential steps.
func compareFileLinesWithMatchers(expected, actual []string) []difference {
	var diffs []difference

	// Parse expected lines into line matchers
	parsedLines := make([]*lineMatcher, len(expected))

	for i, line := range expected {
		parsed, err := parseLine(line)
		if err != nil {
			// Treat parse errors as comparison failures
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: line,
				Actual:   err.Error(),
				Type:     diffChanged,
			})

			continue
		}

		parsedLines[i] = parsed
	}

	maxLines := max(len(expected), len(actual))

	for i := range maxLines {
		var hasExp, hasAct bool

		var expLine *lineMatcher

		var actLine string

		if i < len(parsedLines) && parsedLines[i] != nil {
			expLine = parsedLines[i]
			hasExp = true
		}

		if i < len(actual) {
			actLine = actual[i]
			hasAct = true
		}

		if !hasExp && hasAct {
			// Extra line in actual
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: nil,
				Actual:   actLine,
				Type:     diffAdded,
			})

			continue
		}

		if hasExp && !hasAct {
			// Missing line in actual
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine.original,
				Actual:   nil,
				Type:     diffRemoved,
			})

			continue
		}

		if !hasExp && !hasAct {
			continue
		}

		// Both lines exist - compare them
		if expLine.pattern == nil {
			// Exact match mode
			if expLine.original != actLine {
				diffs = append(diffs, difference{
					Path:     lineNumberPath(i + 1),
					Expected: expLine.original,
					Actual:   actLine,
					Type:     diffChanged,
				})
			}
		} else {
			// Pattern match mode
			if !expLine.pattern.MatchString(actLine) {
				diffs = append(diffs, difference{
					Path:     lineNumberPath(i + 1),
					Expected: expLine.original,
					Actual:   actLine,
					Type:     diffMatcherFailed,
				})
			}
		}
	}

	return diffs
}
