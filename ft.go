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

// Short is a reserved tag that is bidirectionally synced with testing.Short().
// If -test.short is set, Has(Short) reports true. If -ft short is passed,
// the test.short flag is also set to true so testing.Short() reports true.
const Short Tag = "short"

// All is a sentinel tag. -ft all enables every tag: Has returns true for any
// tag and NeedAll never skips. If combined with explicit tags (e.g.
// -ft unit,all), the explicit tags take precedence and "all" is dropped,
// letting you narrow down from a default "all".
const All Tag = "all"

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
		if _, allSet := selected[All]; allSet && len(selected) > 1 {
			delete(selected, All)
		}
		_, shortInSet := selected[Short]
		_, allInSet := selected[All]
		if shortInSet || allInSet {
			if f := flag.Lookup("test.short"); f != nil {
				_ = f.Value.Set("true")
			}
		}
	})
	return selected
}

// Has reports whether the given tag was enabled via -ft.
// Has returns true for every tag if -ft all is set.
// Has(Short) also returns true when testing.Short() is true.
func Has(tag Tag) bool {
	set := selectedSet()
	if _, ok := set[All]; ok {
		return true
	}
	if _, ok := set[tag]; ok {
		return true
	}
	if tag == Short && testing.Short() {
		return true
	}
	return false
}

// NeedAll skips the test unless every provided tag was enabled via -ft.
// Calling NeedAll with no tags is a no-op. The Short tag is also satisfied
// by -test.short (see Has).
func NeedAll(t testing.TB, tags ...Tag) {
	t.Helper()
	var missing []string
	for _, tag := range tags {
		if !Has(tag) {
			missing = append(missing, string(tag))
		}
	}
	if len(missing) > 0 {
		t.Skipf("ft: skipping; missing required tag(s): %s", strings.Join(missing, ", "))
	}
}
