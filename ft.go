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
// Forbid a tag (overrides all): go test ./... -ft all,!postgres
//
// Tags can also come from the FT environment variable:
//
//	FT=integration,app1 go test ./...
//
// The env var makes ft usable across a whole repo: -ft is registered
// per test binary, so passing it to `go test ./...` fails for packages
// that never import ft, while FT reaches only the binaries that look
// for it. A non-empty -ft flag takes precedence over FT.
package ft

import (
	"flag"
	"os"
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
// letting you narrow down from a default "all". Forbidden tags (!tag) always
// override "all".
const All Tag = "all"

var (
	flagValue string

	parseOnce sync.Once
	selected  map[Tag]struct{}
	forbidden map[Tag]struct{}
)

func init() {
	flag.StringVar(&flagValue, "ft", "", "comma-separated list of ft test tags; prefix with ! to forbid (e.g. -ft all,!postgres)")
}

func parse() {
	parseOnce.Do(func() {
		raw := flagValue
		if raw == "" {
			raw = os.Getenv("FT")
		}
		selected = make(map[Tag]struct{})
		forbidden = make(map[Tag]struct{})
		for _, part := range strings.Split(raw, ",") {
			t := strings.TrimSpace(part)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "!") {
				name := strings.TrimSpace(t[1:])
				if name == "" {
					continue
				}
				forbidden[Tag(name)] = struct{}{}
			} else {
				selected[Tag(t)] = struct{}{}
			}
		}
		// Forbidden tags don't count as explicit tags for the override check.
		if _, allSet := selected[All]; allSet && len(selected) > 1 {
			delete(selected, All)
		}
		_, shortInSet := selected[Short]
		_, allInSet := selected[All]
		_, shortForbidden := forbidden[Short]
		if (shortInSet || allInSet) && !shortForbidden {
			if f := flag.Lookup("test.short"); f != nil {
				_ = f.Value.Set("true")
			}
		}
	})
}

// Has reports whether the given tag was enabled via -ft.
// A forbidden tag (!tag) always returns false, overriding everything.
// Has returns true for every tag if -ft all is set (and the tag isn't forbidden).
// Has(Short) also returns true when testing.Short() is true.
func Has(tag Tag) bool {
	parse()
	if _, ok := forbidden[tag]; ok {
		return false
	}
	if _, ok := selected[All]; ok {
		return true
	}
	if _, ok := selected[tag]; ok {
		return true
	}
	if tag == Short && testing.Short() {
		return true
	}
	return false
}

// NeedAll skips the test unless every provided tag was enabled via -ft.
// Calling NeedAll with no tags is a no-op. The Short tag is also satisfied
// by -test.short (see Has). A forbidden tag (!tag) causes an immediate skip.
func NeedAll(t testing.TB, tags ...Tag) {
	t.Helper()
	parse()
	var forbid, missing []string
	for _, tag := range tags {
		if _, ok := forbidden[tag]; ok {
			forbid = append(forbid, string(tag))
		} else if !Has(tag) {
			missing = append(missing, string(tag))
		}
	}
	var reasons []string
	if len(forbid) > 0 {
		reasons = append(reasons, "forbidden tag(s): "+strings.Join(forbid, ", "))
	}
	if len(missing) > 0 {
		reasons = append(reasons, "missing required tag(s): "+strings.Join(missing, ", "))
	}
	if len(reasons) > 0 {
		t.Skipf("ft: skipping; %s", strings.Join(reasons, "; "))
	}
}
