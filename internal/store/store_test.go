package store

import (
	"testing"
)

func TestErrNotFound(t *testing.T) {
	if ErrNotFound == nil {
		t.Fatal("ErrNotFound is nil")
	}
	if ErrNotFound.Error() != "not found" {
		t.Errorf("ErrNotFound.Error() = %q, want %q", ErrNotFound.Error(), "not found")
	}
}

func TestNewStore_InvalidURL(t *testing.T) {
	_, err := NewStore(nil, "not-a-valid-url")
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}
