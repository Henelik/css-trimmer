package diff

import (
	"strings"
	"testing"

	"github.com/Henelik/css-trimmer/internal/config"
)

func TestCompute_BasicUnusedAndUsed(t *testing.T) {
	cfg := &config.Config{}
	inventory := []string{"used", "unused"}
	usedClasses := []string{"used"}

	result, err := Compute(inventory, usedClasses, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result.Used, "used")
	assertContains(t, result.Unused, "unused")
	assertContains(t, result.ToRemove, "unused")
	assertNotContains(t, result.ToRemove, "used")
}

func TestCompute_Whitelist(t *testing.T) {
	cfg := &config.Config{
		Whitelist: []string{"keep-*"},
	}
	inventory := []string{"keep-one", "remove-one"}
	usedClasses := []string{}

	result, err := Compute(inventory, usedClasses, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result.Whitelisted, "keep-one")
	assertContains(t, result.ToRemove, "remove-one")
	assertNotContains(t, result.ToRemove, "keep-one")
}

func TestCompute_Blacklist(t *testing.T) {
	cfg := &config.Config{
		Blacklist: []string{"drop-*"},
	}
	inventory := []string{"drop-one", "keep-one"}
	usedClasses := []string{"keep-one", "drop-one"}

	result, err := Compute(inventory, usedClasses, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result.Blacklisted, "drop-one")
	assertContains(t, result.ToRemove, "drop-one")
	assertNotContains(t, result.ToRemove, "keep-one")
}

func TestCompute_DynamicPatternPrecompiled(t *testing.T) {
	cfg := &config.Config{
		DynamicClassPatterns: []string{`^btn-.*$`, `^col-(sm|md|lg)-\d+$`},
	}
	inventory := []string{"btn-primary", "btn-secondary", "col-sm-3", "plain"}
	usedClasses := []string{}

	result, err := Compute(inventory, usedClasses, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result.Used, "btn-primary")
	assertContains(t, result.Used, "btn-secondary")
	assertContains(t, result.Used, "col-sm-3")
	assertContains(t, result.Unused, "plain")
	assertNotContains(t, result.Used, "plain")
}

func TestCompute_InvalidDynamicPattern(t *testing.T) {
	cfg := &config.Config{
		DynamicClassPatterns: []string{"[invalid"},
	}

	_, err := Compute([]string{"foo"}, []string{}, cfg)
	if err == nil {
		t.Fatalf("expected error for invalid dynamic pattern, got nil")
	}
	if !strings.Contains(err.Error(), "invalid dynamic class pattern") {
		t.Errorf("expected error to mention invalid dynamic class pattern, got: %v", err)
	}
}

func TestCompute_InvalidWhitelistGlob(t *testing.T) {
	cfg := &config.Config{
		Whitelist: []string{"["},
	}

	_, err := Compute([]string{"foo"}, []string{}, cfg)
	if err == nil {
		t.Fatalf("expected error for invalid whitelist glob, got nil")
	}
	if !strings.Contains(err.Error(), "invalid whitelist pattern") {
		t.Errorf("expected error to mention invalid whitelist pattern, got: %v", err)
	}
}

func TestCompute_InvalidBlacklistGlob(t *testing.T) {
	cfg := &config.Config{
		Blacklist: []string{"bad[}"},
	}

	_, err := Compute([]string{"foo"}, []string{}, cfg)
	if err == nil {
		t.Fatalf("expected error for invalid blacklist glob, got nil")
	}
	if !strings.Contains(err.Error(), "invalid blacklist pattern") {
		t.Errorf("expected error to mention invalid blacklist pattern, got: %v", err)
	}
}

func TestCompute_EmptyPatterns(t *testing.T) {
	cfg := &config.Config{}
	inventory := []string{"a", "b"}
	usedClasses := []string{"a"}

	result, err := Compute(inventory, usedClasses, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result.Used, "a")
	assertContains(t, result.Unused, "b")
}

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("expected %v to contain %q", slice, item)
}

func assertNotContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			t.Errorf("expected %v not to contain %q", slice, item)
			return
		}
	}
}
