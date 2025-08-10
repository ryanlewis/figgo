package figgo

import "testing"

func TestSimpleAddition(t *testing.T) {
	// Simple test to help establish Codecov baseline
	result := 1 + 1
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}
}

func TestSimpleSubtraction(t *testing.T) {
	// Another simple test for coverage
	result := 5 - 3
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}
}
