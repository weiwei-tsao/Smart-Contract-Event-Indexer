package main

import "testing"

func TestMaskConnectionURL(t *testing.T) {
	raw := "postgres://user:secret@localhost:5432/db"
	masked := maskConnectionURL(raw)
	if masked == raw {
		t.Fatalf("expected masked URL, got original")
	}
	if masked == "" {
		t.Fatalf("expected non-empty masked URL")
	}

	// Malformed input should not panic and should return something meaningful
	_ = maskConnectionURL("://totally invalid")
}
