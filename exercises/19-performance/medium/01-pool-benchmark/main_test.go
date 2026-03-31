package pool_benchmark

import (
	"encoding/json"
	"testing"
)

var testData = []byte(`{"name": "Alice", "age": 30, "items": [1, 2, 3]}`)

func TestProcessWithAlloc(t *testing.T) {
	result := ProcessWithAlloc(testData)
	if result == nil {
		t.Fatal("nil result")
	}
	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["processed"] != true {
		t.Error("processed field not set")
	}
	if m["name"] != "Alice" {
		t.Error("original data lost")
	}
}

func TestProcessWithPool(t *testing.T) {
	result := ProcessWithPool(testData)
	if result == nil {
		t.Fatal("nil result")
	}
	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["processed"] != true {
		t.Error("processed field not set")
	}
}

func BenchmarkProcessWithAlloc(b *testing.B) {
	for b.Loop() {
		ProcessWithAlloc(testData)
	}
}

func BenchmarkProcessWithPool(b *testing.B) {
	for b.Loop() {
		ProcessWithPool(testData)
	}
}
