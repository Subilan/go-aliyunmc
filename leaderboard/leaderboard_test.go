package leaderboard

import "testing"

func TestSortEntries(t *testing.T) {
	entries := []Entry{
		{UUID: "b", Value: 10},
		{UUID: "a", Value: 30},
		{UUID: "d", Value: 20},
		{UUID: "c", Value: 20},
	}

	desc := sortEntries(entries, "")
	wantDesc := []string{"a", "c", "d", "b"}
	for i, uuid := range wantDesc {
		if desc[i].UUID != uuid {
			t.Fatalf("desc order mismatch at %d: got %s want %s", i, desc[i].UUID, uuid)
		}
	}

	asc := sortEntries(entries, "asc")
	wantAsc := []string{"b", "c", "d", "a"}
	for i, uuid := range wantAsc {
		if asc[i].UUID != uuid {
			t.Fatalf("asc order mismatch at %d: got %s want %s", i, asc[i].UUID, uuid)
		}
	}

	if entries[0].UUID != "b" {
		t.Fatalf("sortEntries must not modify its input, got %s first", entries[0].UUID)
	}
}

func TestSortEntriesEmpty(t *testing.T) {
	sorted := sortEntries(nil, "")
	if sorted == nil {
		t.Fatal("expected an empty slice instead of nil")
	}
	if len(sorted) != 0 {
		t.Fatalf("expected an empty slice, got %d entries", len(sorted))
	}
}
