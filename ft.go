// Package ft provides selective test execution via tags driven by a command-line flag.
//
// Define tags as package-level variables and gate tests on them:
//
//	var (
//	    TagIntegration = ft.Tag("integration")
//	    TagApp1        = ft.Tag("app1")
//	)
//
//	func TestThing(t *testing.T) {
//	    ft.NeedAll(t, TagIntegration, TagApp1)
//	    // ...
//	}
//
// Run with: go test ./... -ft integration,app1
package ft

import (
	"flag"
	"strings"
	"sync"
	"testing"
)

// Tag is a name used to selectively include tests via the -ft flag.
type Tag string

var (
	flagValue string

	parseOnce sync.Once
	selected  map[Tag]struct{}
)

func init() {
	flag.StringVar(&flagValue, "ft", "", "comma-separated list of ft test tags to enable")
}

func selectedSet() map[Tag]struct{} {
	parseOnce.Do(func() {
		selected = make(map[Tag]struct{})
		for _, raw := range strings.Split(flagValue, ",") {
			t := strings.TrimSpace(raw)
			if t == "" {
				continue
			}
			selected[Tag(t)] = struct{}{}
		}
	})
	return selected
}

// Has reports whether the given tag was enabled via the -ft flag.
func Has(tag Tag) bool {
	_, ok := selectedSet()[tag]
	return ok
}

// NeedAll skips the test unless every provided tag was enabled via -ft.
// Calling NeedAll with no tags is a no-op.
func NeedAll(t testing.TB, tags ...Tag) {
	t.Helper()
	set := selectedSet()
	var missing []string
	for _, tag := range tags {
		if _, ok := set[tag]; !ok {
			missing = append(missing, string(tag))
		}
	}
	if len(missing) > 0 {
		t.Skipf("ft: skipping; missing required tag(s): %s", strings.Join(missing, ", "))
	}
}
