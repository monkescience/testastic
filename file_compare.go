package testastic

import "fmt"

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
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: nil,
				Actual:   actLine,
				Type:     diffAdded,
			})

			continue
		}

		if !hasAct {
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: expLine,
				Actual:   nil,
				Type:     diffRemoved,
			})

			continue
		}

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

// compareFileLinesWithMatchers compares expected and actual lines, supporting
// matchers. It returns an error if an expected line contains an invalid or
// unknown matcher, which is caller misuse rather than a comparison result.
func compareFileLinesWithMatchers(expected, actual []string) ([]difference, error) {
	parsedLines := make([]*lineMatcher, len(expected))

	for i, line := range expected {
		parsed, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", lineNumberPath(i+1), err)
		}

		parsedLines[i] = parsed
	}

	var diffs []difference

	maxLines := max(len(expected), len(actual))

	for i := range maxLines {
		switch {
		case i >= len(parsedLines):
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: nil,
				Actual:   actual[i],
				Type:     diffAdded,
			})
		case i >= len(actual):
			diffs = append(diffs, difference{
				Path:     lineNumberPath(i + 1),
				Expected: parsedLines[i].original,
				Actual:   nil,
				Type:     diffRemoved,
			})
		default:
			diffs = appendLineDiff(diffs, parsedLines[i], actual[i], i+1)
		}
	}

	return diffs, nil
}

func appendLineDiff(diffs []difference, expLine *lineMatcher, actLine string, lineNum int) []difference {
	if expLine.pattern == nil {
		if expLine.original != actLine {
			diffs = append(diffs, difference{
				Path:     lineNumberPath(lineNum),
				Expected: expLine.original,
				Actual:   actLine,
				Type:     diffChanged,
			})
		}

		return diffs
	}

	if !lineMatches(expLine, actLine) {
		diffs = append(diffs, difference{
			Path:     lineNumberPath(lineNum),
			Expected: expLine.original,
			Actual:   actLine,
			Type:     diffMatcherFailed,
		})
	}

	return diffs
}

// lineMatches reports whether actLine satisfies a matcher line: the line regex
// matches and every captured group is accepted by its corresponding matcher.
// Running Match (not just the approximating regex) is what makes custom and
// strict matchers actually validate the captured value.
func lineMatches(line *lineMatcher, actLine string) bool {
	submatches := line.pattern.FindStringSubmatch(actLine)
	if submatches == nil {
		return false
	}

	if len(line.captureIndexes) != len(line.matchers) {
		return false
	}

	for i, m := range line.matchers {
		captureIndex := line.captureIndexes[i]
		if captureIndex >= len(submatches) || !matchStringMatcher(m, submatches[captureIndex]) {
			return false
		}
	}

	return true
}
