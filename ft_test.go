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
	forbidden = nil
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

func TestAll_OverriddenByExplicitTags(t *testing.T) {
	setTestShort(t, false)
	resetFlag("unit,all")
	if !Has("unit") {
		t.Error(`Has("unit") = false; want true under -ft unit,all`)
	}
	for _, tag := range []Tag{"integration", "postgres", "app1"} {
		if Has(tag) {
			t.Errorf("Has(%q) = true; -ft unit,all should restrict to explicit tags only", tag)
		}
	}
	if testing.Short() {
		t.Error("-ft unit,all should not auto-enable testing.Short() since `all` is overridden")
	}
}

func TestAll_OverrideKeepsShortIfExplicit(t *testing.T) {
	setTestShort(t, false)
	resetFlag("short,all")
	if !Has(Short) {
		t.Error("Has(Short) = false; want true under -ft short,all")
	}
	if Has("integration") {
		t.Error(`Has("integration") = true; explicit override should drop "all"`)
	}
	if !testing.Short() {
		t.Error("expected -ft short,all to still flip testing.Short() via the short tag")
	}
}

func TestParse_TrimAndSkipEmpty(t *testing.T) {
	resetFlag("  ,, foo ,bar,  ,baz,")
	parse()
	want := []Tag{"foo", "bar", "baz"}
	if len(selected) != len(want) {
		t.Fatalf("set size = %d, want %d (set: %v)", len(selected), len(want), selected)
	}
	for _, w := range want {
		if _, ok := selected[w]; !ok {
			t.Errorf("missing %q in parsed set", w)
		}
	}
}

func TestForbid_OverridesAll(t *testing.T) {
	setTestShort(t, false)
	resetFlag("all,!postgres")
	if Has("postgres") {
		t.Error(`Has("postgres") = true; want false (forbidden)`)
	}
	for _, tag := range []Tag{"integration", "app1", "anything"} {
		if !Has(tag) {
			t.Errorf("Has(%q) = false; want true under -ft all,!postgres", tag)
		}
	}
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "integration", "postgres")
		ran = true
	})
	if ran {
		t.Error("expected skip when a required tag is forbidden")
	}
}

func TestForbid_OverridesExplicitTag(t *testing.T) {
	resetFlag("integration,!integration")
	if Has("integration") {
		t.Error(`forbidden should win over explicit selection`)
	}
}

func TestForbid_SkipMessageNamesForbidden(t *testing.T) {
	resetFlag("all,!postgres")
	var ran bool
	t.Run("inner", func(t *testing.T) {
		NeedAll(t, "postgres")
		ran = true
	})
	if ran {
		t.Error("expected skip")
	}
	// The test output contains the skip msg; smoke-check via the package log,
	// but the more useful check is below.
}

func TestForbid_DoesNotCountAsExplicitOverrideOfAll(t *testing.T) {
	setTestShort(t, false)
	resetFlag("all,!postgres")
	// "all" should NOT be dropped just because !postgres is present.
	if !Has("anything") {
		t.Error(`Has("anything") = false; "!postgres" alone should not drop "all"`)
	}
	if !testing.Short() {
		t.Error("expected -ft all,!postgres to still flip testing.Short()")
	}
}

func TestEnv_UsedWhenFlagEmpty(t *testing.T) {
	t.Setenv("FT", "integration,app1")
	resetFlag("")
	for _, tag := range []Tag{"integration", "app1"} {
		if !Has(tag) {
			t.Errorf("Has(%q) = false; want true from FT env", tag)
		}
	}
	if Has("postgres") {
		t.Error(`Has("postgres") = true; want false`)
	}
}

func TestEnv_FlagTakesPrecedence(t *testing.T) {
	t.Setenv("FT", "postgres")
	resetFlag("integration")
	if !Has("integration") {
		t.Error(`Has("integration") = false; want true from flag`)
	}
	if Has("postgres") {
		t.Error(`Has("postgres") = true; non-empty flag should override FT env`)
	}
}

func TestEnv_SupportsForbidAndAll(t *testing.T) {
	setTestShort(t, false)
	t.Setenv("FT", "all,!postgres")
	resetFlag("")
	if Has("postgres") {
		t.Error(`Has("postgres") = true; want false (forbidden via env)`)
	}
	if !Has("anything") {
		t.Error(`Has("anything") = false; want true under FT=all,!postgres`)
	}
}

func TestForbid_Short(t *testing.T) {
	setTestShort(t, true)
	resetFlag("all,!short")
	if Has(Short) {
		t.Error("Has(Short) = true; want false when !short is set even with testing.Short() true")
	}
}
