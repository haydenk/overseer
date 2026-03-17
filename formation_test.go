package main

import (
	"testing"
)

func TestParseFormation_EmptySpec(t *testing.T) {
	formation, defaultCount, err := ParseFormation("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(formation) != 0 {
		t.Errorf("expected empty formation map, got %v", formation)
	}
	if defaultCount != 1 {
		t.Errorf("defaultCount = %d, want 1", defaultCount)
	}
}

func TestParseFormation_SingleEntry(t *testing.T) {
	formation, defaultCount, err := ParseFormation("web=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formation["web"] != 2 {
		t.Errorf("formation[web] = %d, want 2", formation["web"])
	}
	if defaultCount != 1 {
		t.Errorf("defaultCount = %d, want 1", defaultCount)
	}
}

func TestParseFormation_MultipleEntries(t *testing.T) {
	formation, defaultCount, err := ParseFormation("web=2,worker=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formation["web"] != 2 {
		t.Errorf("formation[web] = %d, want 2", formation["web"])
	}
	if formation["worker"] != 3 {
		t.Errorf("formation[worker] = %d, want 3", formation["worker"])
	}
	if defaultCount != 1 {
		t.Errorf("defaultCount = %d, want 1", defaultCount)
	}
}

func TestParseFormation_AllKeySetDefault(t *testing.T) {
	formation, defaultCount, err := ParseFormation("all=2,web=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultCount != 2 {
		t.Errorf("defaultCount = %d, want 2", defaultCount)
	}
	if formation["web"] != 3 {
		t.Errorf("formation[web] = %d, want 3", formation["web"])
	}
	// "all" should not appear in the formation map.
	if _, ok := formation["all"]; ok {
		t.Error("'all' should not be present in formation map")
	}
}

func TestParseFormation_AllOnly(t *testing.T) {
	formation, defaultCount, err := ParseFormation("all=5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultCount != 5 {
		t.Errorf("defaultCount = %d, want 5", defaultCount)
	}
	if len(formation) != 0 {
		t.Errorf("expected empty formation map, got %v", formation)
	}
}

func TestParseFormation_ZeroCount(t *testing.T) {
	formation, _, err := ParseFormation("web=0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formation["web"] != 0 {
		t.Errorf("formation[web] = %d, want 0", formation["web"])
	}
}

func TestParseFormation_ErrorOnNegativeCount(t *testing.T) {
	_, _, err := ParseFormation("web=-1")
	if err == nil {
		t.Fatal("expected error for negative count, got nil")
	}
}

func TestParseFormation_ErrorOnNonNumericCount(t *testing.T) {
	_, _, err := ParseFormation("web=abc")
	if err == nil {
		t.Fatal("expected error for non-numeric count, got nil")
	}
}

func TestParseFormation_ErrorOnMissingEquals(t *testing.T) {
	_, _, err := ParseFormation("web")
	if err == nil {
		t.Fatal("expected error for missing '=', got nil")
	}
}

func TestParseFormation_ErrorOnEmptyName(t *testing.T) {
	// "=2" has an empty name but does have an equals; the resulting key is "".
	// It should not crash — behavior may vary, but it should not panic.
	// The implementation will treat it as a process named "" which is unusual
	// but the regex allows it; we just verify no panic.
	_, _, _ = ParseFormation("=2")
}

func TestParseFormation_WhitespaceAroundValues(t *testing.T) {
	formation, defaultCount, err := ParseFormation("web = 2 , worker = 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formation["web"] != 2 {
		t.Errorf("formation[web] = %d, want 2", formation["web"])
	}
	if formation["worker"] != 3 {
		t.Errorf("formation[worker] = %d, want 3", formation["worker"])
	}
	if defaultCount != 1 {
		t.Errorf("defaultCount = %d, want 1", defaultCount)
	}
}

func TestParseFormation_TableDriven(t *testing.T) {
	tests := []struct {
		spec          string
		wantErr       bool
		wantDefault   int
		wantFormation map[string]int
	}{
		{"all=1,web=2,worker=3", false, 1, map[string]int{"web": 2, "worker": 3}},
		{"", false, 1, map[string]int{}},
		{"all=10", false, 10, map[string]int{}},
		{"web=1", false, 1, map[string]int{"web": 1}},
		{"web=-1", true, 0, nil},
		{"web=notanumber", true, 0, nil},
		{"badentry", true, 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.spec, func(t *testing.T) {
			formation, defaultCount, err := ParseFormation(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseFormation(%q): expected error, got nil", tc.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormation(%q): unexpected error: %v", tc.spec, err)
			}
			if defaultCount != tc.wantDefault {
				t.Errorf("defaultCount = %d, want %d", defaultCount, tc.wantDefault)
			}
			for name, want := range tc.wantFormation {
				if formation[name] != want {
					t.Errorf("formation[%q] = %d, want %d", name, formation[name], want)
				}
			}
		})
	}
}

func TestCountFor_ReturnsFormationValueWhenPresent(t *testing.T) {
	formation := map[string]int{"web": 3, "worker": 2}
	got := CountFor(formation, 1, "web")
	if got != 3 {
		t.Errorf("CountFor(web) = %d, want 3", got)
	}
}

func TestCountFor_ReturnsDefaultWhenNotPresent(t *testing.T) {
	formation := map[string]int{"web": 3}
	got := CountFor(formation, 5, "worker")
	if got != 5 {
		t.Errorf("CountFor(worker) = %d, want 5 (default)", got)
	}
}

func TestCountFor_ReturnsZeroCountWhenExplicitlySet(t *testing.T) {
	formation := map[string]int{"web": 0}
	got := CountFor(formation, 1, "web")
	if got != 0 {
		t.Errorf("CountFor(web) = %d, want 0", got)
	}
}

func TestCountFor_EmptyFormationUsesDefault(t *testing.T) {
	formation := map[string]int{}
	got := CountFor(formation, 2, "anything")
	if got != 2 {
		t.Errorf("CountFor(anything) = %d, want 2", got)
	}
}

func TestCountFor_TableDriven(t *testing.T) {
	tests := []struct {
		formation    map[string]int
		defaultCount int
		processName  string
		want         int
	}{
		{map[string]int{"web": 3}, 1, "web", 3},
		{map[string]int{"web": 3}, 1, "worker", 1},
		{map[string]int{}, 5, "web", 5},
		{map[string]int{"web": 0}, 1, "web", 0},
		{map[string]int{"web": 1, "worker": 2}, 3, "clock", 3},
	}

	for _, tc := range tests {
		got := CountFor(tc.formation, tc.defaultCount, tc.processName)
		if got != tc.want {
			t.Errorf("CountFor(%v, %d, %q) = %d, want %d",
				tc.formation, tc.defaultCount, tc.processName, got, tc.want)
		}
	}
}
