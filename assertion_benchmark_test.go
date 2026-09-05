package testastic_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func BenchmarkFileAssertions(b *testing.B) {
	formats := []struct {
		name   string
		prefix string
		item   string
		suffix string
		assert func(testing.TB, string, []byte, ...testastic.Option)
	}{
		{"JSON", "[", `{"name":"value","active":true},`, "]", testastic.AssertJSON[[]byte]},
		{"YAML", "", "- name: value\n  active: true\n", "", testastic.AssertYAML[[]byte]},
		{"HTML", "<ul>", "<li>value</li>", "</ul>", testastic.AssertHTML[[]byte]},
		{"Text", "", "name=value active=true\n", "", testastic.AssertFile[[]byte]},
	}

	for _, format := range formats {
		b.Run(format.name, func(b *testing.B) {
			for _, size := range []int{16, 256} {
				b.Run(strconv.Itoa(size), func(b *testing.B) {
					body := strings.Repeat(format.item, size)
					if format.name == "JSON" {
						body = strings.TrimSuffix(body, ",")
					}

					actual := []byte(format.prefix + body + format.suffix)
					path := filepath.Join(b.TempDir(), "expected")

					err := os.WriteFile(path, actual, 0o600)
					if err != nil {
						b.Fatal(err)
					}

					format.assert(b, path, actual)
					b.ReportAllocs()
					b.SetBytes(int64(len(actual)))
					b.ResetTimer()

					for b.Loop() {
						format.assert(b, path, actual)
					}
				})
			}
		})
	}
}
