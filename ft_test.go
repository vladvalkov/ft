package ft

import (
	"flag"
	"sync"
	"testing"
)

func resetFlag(raw string) {
	flagValue = raw
	parseOnce = sync.Once{}
	selected = nil
}

// setTestShort overrides -test.short for the duration of the test and restores it after.
func setTestShort(t *testing.T, v bool) {
	t.Helper()
	f := flag.Lookup("test.short")
	if f == nil {
		t.Skip("test.short flag not registered")
	}
	orig := f.Value.String()
	t.Cleanup(func() { _ = f.Value.Set(orig) })
	val := "false"
	if v {
		val = "true"
	}
	if err := f.Value.Set(val); err != nil {
		t.Fatal(err)
	}
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

func TestShort_FromTestingFlag(t *testing.T) {
	setTestShort(t, true)
	resetFlag("")
	if !Has(Short) {
		t.Error("Has(Short) = false; want true when testing.Short() is true")
	}
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, Short)
		ran = true
	})
	if !ran {
		t.Error("NeedAll(Short) should not skip when testing.Short() is true")
	}
}

func TestShort_FromFTFlagSyncsTestingShort(t *testing.T) {
	setTestShort(t, false)
	if testing.Short() {
		t.Fatal("setup: expected testing.Short() to be false")
	}
	resetFlag("short")
	if !Has(Short) {
		t.Error("Has(Short) = false; want true when -ft short is set")
	}
	if !testing.Short() {
		t.Error("expected -ft short to set testing.Short() to true")
	}
}

func TestShort_NotPresent(t *testing.T) {
	setTestShort(t, false)
	resetFlag("integration")
	if Has(Short) {
		t.Error("Has(Short) = true; want false (neither -test.short nor -ft short set)")
	}
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, Short)
		ran = true
	})
	if ran {
		t.Error("NeedAll(Short) should skip when short is not enabled")
	}
}

func TestAll_Sentinel(t *testing.T) {
	setTestShort(t, false)
	resetFlag("all")
	for _, tag := range []Tag{"integration", "postgres", "anything", "app1"} {
		if !Has(tag) {
			t.Errorf("Has(%q) = false; want true under -ft all", tag)
		}
	}
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "integration", "postgres", "made-up")
		ran = true
	})
	if !ran {
		t.Error("NeedAll should not skip under -ft all")
	}
	if !testing.Short() {
		t.Error("-ft all should also enable testing.Short()")
	}
}

func TestAll_MixedWithOtherTags(t *testing.T) {
	setTestShort(t, false)
	resetFlag("integration,all")
	if !Has("postgres") {
		t.Error("-ft integration,all should still satisfy unrelated tags")
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
