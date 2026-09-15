// Package player — regression tests for the quality-selection UI.
//
// 2026-04-28: ExtractVideoSourcesWithPrompt had three UX bugs:
//
//  1. When two sources shared a numeric quality (e.g. two 720p mirrors), the
//     post-prompt loop matched on the rendered label string and always
//     returned the *first* matching source. The user's actual selection was
//     silently dropped.
//  2. Pressing Esc/Ctrl-C (fuzzyfinder.ErrAbort) silently played
//     sources[0].URL with a nil error — no way to back out.
//  3. Sources with Quality == 0 (label digits unparseable) rendered as "0p",
//     and the order matched whatever the upstream returned (often 360p first
//     when 1080p was available).
//
// The fix extracted the pure logic into buildQualityMenu so it could be
// tested without driving the interactive fuzzyfinder. These tests pin the
// new contract: descending sort, "Auto" fallback for unknown resolution,
// "(mirror N)" suffix on duplicates, and 1:1 alignment between labels and
// the returned slice (so caller indexing is unambiguous).
package player

import (
	"strings"
	"testing"
)

type qualitySource = struct {
	Quality int
	URL     string
}

func TestBuildQualityMenu_SortsDescendingByQuality(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 360, URL: "https://cdn/360.mp4"},
		{Quality: 1080, URL: "https://cdn/1080.mp4"},
		{Quality: 720, URL: "https://cdn/720.mp4"},
	}
	sorted, items := buildQualityMenu(in)

	if len(sorted) != 3 || len(items) != 3 {
		t.Fatalf("expected 3 entries, got sorted=%d items=%d", len(sorted), len(items))
	}
	wantOrder := []int{1080, 720, 360}
	for i, want := range wantOrder {
		if sorted[i].Quality != want {
			t.Fatalf("sorted[%d].Quality = %d, want %d", i, sorted[i].Quality, want)
		}
	}
	wantLabels := []string{"1080p", "720p", "360p"}
	for i, want := range wantLabels {
		if items[i] != want {
			t.Fatalf("items[%d] = %q, want %q", i, items[i], want)
		}
	}
}

// Bug 1 regression: duplicate qualities must produce distinct labels so the
// fuzzyfinder index aligns 1:1 with the sorted slice. Without the
// "(mirror N)" suffix, two identical "720p" labels would let the previous
// string-match loop return the wrong source.
func TestBuildQualityMenu_DisambiguatesDuplicateQualities(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 720, URL: "https://mirror-a/720.mp4"},
		{Quality: 720, URL: "https://mirror-b/720.mp4"},
		{Quality: 720, URL: "https://mirror-c/720.mp4"},
	}
	sorted, items := buildQualityMenu(in)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	uniq := map[string]struct{}{}
	for _, l := range items {
		uniq[l] = struct{}{}
	}
	if len(uniq) != 3 {
		t.Fatalf("labels must be unique to avoid selection ambiguity, got %v", items)
	}

	if items[0] != "720p" {
		t.Fatalf("first duplicate must keep the bare label; got %q", items[0])
	}
	for i := 1; i < 3; i++ {
		if !strings.Contains(items[i], "mirror") {
			t.Fatalf("items[%d] must include 'mirror' suffix; got %q", i, items[i])
		}
	}

	// Stable order must preserve URL identity per index — the fix relies on
	// indexing directly into the sorted slice with the user's chosen idx.
	if sorted[0].URL != "https://mirror-a/720.mp4" {
		t.Fatalf("sorted[0].URL = %q, want mirror-a (stable sort)", sorted[0].URL)
	}
	if sorted[1].URL != "https://mirror-b/720.mp4" {
		t.Fatalf("sorted[1].URL = %q, want mirror-b", sorted[1].URL)
	}
	if sorted[2].URL != "https://mirror-c/720.mp4" {
		t.Fatalf("sorted[2].URL = %q, want mirror-c", sorted[2].URL)
	}
}

// Bug 3 regression: a source whose label digits could not be parsed used to
// render as "0p", which is meaningless to the user. It must now render as
// "Auto" so the menu is informative.
func TestBuildQualityMenu_UnknownQualityRendersAsAuto(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 0, URL: "https://cdn/unknown.mp4"},
		{Quality: 720, URL: "https://cdn/720.mp4"},
	}
	_, items := buildQualityMenu(in)

	// 720p sorts above Auto (0).
	if items[0] != "720p" {
		t.Fatalf("items[0] = %q, want \"720p\"", items[0])
	}
	if items[1] != "Auto" {
		t.Fatalf("items[1] = %q, want \"Auto\" (must not show '0p')", items[1])
	}
	for _, l := range items {
		if l == "0p" {
			t.Fatalf("'0p' must never appear in the rendered menu, got %v", items)
		}
	}
}

// Multiple unknown-quality sources must still disambiguate so the user can
// pick a specific one — same contract as the duplicate-720p case.
func TestBuildQualityMenu_DisambiguatesDuplicateAutoEntries(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 0, URL: "https://a/x.mp4"},
		{Quality: 0, URL: "https://b/x.mp4"},
	}
	_, items := buildQualityMenu(in)

	if items[0] != "Auto" {
		t.Fatalf("items[0] = %q, want \"Auto\"", items[0])
	}
	if !strings.Contains(items[1], "Auto") || !strings.Contains(items[1], "mirror") {
		t.Fatalf("items[1] must disambiguate as Auto + mirror N; got %q", items[1])
	}
	if items[0] == items[1] {
		t.Fatalf("duplicate Auto labels would re-introduce Bug 1; got %v", items)
	}
}

// Lock the alignment invariant the caller depends on: items[i] describes
// sorted[i], and indexing the user's selection idx into sorted yields the
// exact source they picked. This is the structural fix for Bug 1.
func TestBuildQualityMenu_LabelsAlignWithSourcesByIndex(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 1080, URL: "https://cdn/1080.mp4"},
		{Quality: 720, URL: "https://m1/720.mp4"},
		{Quality: 720, URL: "https://m2/720.mp4"},
		{Quality: 0, URL: "https://cdn/auto-a.mp4"},
		{Quality: 0, URL: "https://cdn/auto-b.mp4"},
	}
	sorted, items := buildQualityMenu(in)

	if len(sorted) != len(items) {
		t.Fatalf("len(sorted)=%d must equal len(items)=%d", len(sorted), len(items))
	}
	for i := range sorted {
		// Every label must be unique so the index-based selection in
		// ExtractVideoSourcesWithPrompt is unambiguous.
		for j := i + 1; j < len(items); j++ {
			if items[i] == items[j] {
				t.Fatalf("items[%d] == items[%d] == %q breaks index alignment", i, j, items[i])
			}
		}
	}
}

func TestBuildQualityMenu_EmptyInput(t *testing.T) {
	t.Parallel()

	sorted, items := buildQualityMenu(nil)
	if len(sorted) != 0 || len(items) != 0 {
		t.Fatalf("empty input must yield empty output; got sorted=%v items=%v", sorted, items)
	}
}

func TestBuildQualityMenu_SingleSource(t *testing.T) {
	t.Parallel()

	in := []qualitySource{{Quality: 480, URL: "https://cdn/480.mp4"}}
	sorted, items := buildQualityMenu(in)
	if len(sorted) != 1 || sorted[0].URL != "https://cdn/480.mp4" {
		t.Fatalf("single source must be preserved verbatim; got %v", sorted)
	}
	if len(items) != 1 || items[0] != "480p" {
		t.Fatalf("single source must render as bare label; got %v", items)
	}
}

// Sort must be stable for equal qualities so callers that pin "first
// preferred mirror" semantics keep working.
func TestBuildQualityMenu_StableSortOnTies(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 1080, URL: "https://primary/1080.mp4"},
		{Quality: 1080, URL: "https://backup/1080.mp4"},
		{Quality: 1080, URL: "https://tertiary/1080.mp4"},
	}
	sorted, _ := buildQualityMenu(in)
	wantOrder := []string{
		"https://primary/1080.mp4",
		"https://backup/1080.mp4",
		"https://tertiary/1080.mp4",
	}
	for i, want := range wantOrder {
		if sorted[i].URL != want {
			t.Fatalf("stable sort broken at %d: got %q want %q", i, sorted[i].URL, want)
		}
	}
}

// Caller-side input must not be mutated. The previous implementation
// modified the slice in place via sort.Slice on the input — if a future
// regression reintroduces that, downstream code that reuses the original
// `sources` slice (e.g. for diagnostic logging) would see a different
// order than it provided.
func TestBuildQualityMenu_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := []qualitySource{
		{Quality: 360, URL: "https://cdn/360.mp4"},
		{Quality: 1080, URL: "https://cdn/1080.mp4"},
		{Quality: 720, URL: "https://cdn/720.mp4"},
	}
	snapshot := append([]qualitySource(nil), in...)

	_, _ = buildQualityMenu(in)

	for i := range snapshot {
		if in[i] != snapshot[i] {
			t.Fatalf("input mutated at %d: got %+v want %+v", i, in[i], snapshot[i])
		}
	}
}
