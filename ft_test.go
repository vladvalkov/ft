package ft

import (
	"sync"
	"testing"
)

func resetFlag(raw string) {
	flagValue = raw
	parseOnce = sync.Once{}
	selected = nil
}

func TestHas(t *testing.T) {
	resetFlag("integration, app1 ,, unit")
	for _, tag := range []Tag{"integration", "app1", "unit"} {
		if !Has(tag) {
			t.Errorf("Has(%q) = false; want true", tag)
		}
	}
	if Has("postgres") {
		t.Error(`Has("postgres") = true; want false`)
	}
}

func TestNeedAll_AllPresent(t *testing.T) {
	resetFlag("integration,app1")
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "integration", "app1")
		ran = true
	})
	if !ran {
		t.Error("expected test body to run when all tags present")
	}
}

func TestNeedAll_Missing(t *testing.T) {
	resetFlag("integration")
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "integration", "app1")
		ran = true
	})
	if ran {
		t.Error("expected test body to be skipped when a tag is missing")
	}
}

func TestNeedAll_NoTagsArg(t *testing.T) {
	resetFlag("")
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t)
		ran = true
	})
	if !ran {
		t.Error("NeedAll with no tags should not skip")
	}
}

func TestNeedAll_EmptyFlag(t *testing.T) {
	resetFlag("")
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "integration")
		ran = true
	})
	if ran {
		t.Error("expected skip when no tags are enabled")
	}
}

func TestParse_TrimAndSkipEmpty(t *testing.T) {
	resetFlag("  ,, foo ,bar,  ,baz,")
	set := selectedSet()
	want := []Tag{"foo", "bar", "baz"}
	if len(set) != len(want) {
		t.Fatalf("set size = %d, want %d (set: %v)", len(set), len(want), set)
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Errorf("missing %q in parsed set", w)
		}
	}
}
