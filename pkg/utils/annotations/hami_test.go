package annotations

import (
	"reflect"
	"testing"
)

func TestMarshalGPUAllocationAnnotation_Valid(t *testing.T) {
	// This input is based on the expected format: UUID,Vendor,Memory,Vgpus:...
	input := "uuid1,vendor1,1024,2:uuid2,vendor2,2048,4:;"
	expected := []*GPUAllocation{
		{UUID: "uuid1", Vendor: "vendor1", Memory: 1024, Core: 2},
		{UUID: "uuid2", Vendor: "vendor2", Memory: 2048, Core: 4},
	}

	allocs, err := MarshalGPUAllocationAnnotation(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(allocs, expected) {
		t.Errorf("got %+v, want %+v", allocs, expected)
	}
}

func TestMarshalGPUAllocationAnnotation_InvalidFormat(t *testing.T) {
	// Missing one field (should be 4 fields per allocation)
	input := "uuid1,vendor1,1024:;"
	_, err := MarshalGPUAllocationAnnotation(input)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestMarshalGPUAllocationAnnotation_InvalidMemory(t *testing.T) {
	// Memory is not an integer
	input := "uuid1,vendor1,notanint,2:;"
	_, err := MarshalGPUAllocationAnnotation(input)
	if err == nil {
		t.Fatal("expected error for invalid memory, got nil")
	}
}

func TestMarshalGPUAllocationAnnotation_InvalidVgpus(t *testing.T) {
	// Vgpus is not an integer
	input := "uuid1,vendor1,1024,notanint:;"
	_, err := MarshalGPUAllocationAnnotation(input)
	if err == nil {
		t.Fatal("expected error for invalid vgpus, got nil")
	}
}

func TestMarshalGPUAllocationAnnotation_EmptyInput(t *testing.T) {
	input := ";"
	allocs, err := MarshalGPUAllocationAnnotation(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(allocs) != 0 {
		t.Errorf("expected empty result, got %+v", allocs)
	}
}

func TestUnmarshalGPUAllocationAnnotation(t *testing.T) {
	allocs := []*GPUAllocation{
		{UUID: "uuid1", Vendor: "vendor1", Memory: 1024, Core: 2},
		{UUID: "uuid2", Vendor: "vendor2", Memory: 2048, Core: 4},
	}
	result := UnmarshalGPUAllocationAnnotation(allocs)
	if len(result) == 0 || result[len(result)-1] != ';' {
		t.Errorf("expected result to end with ';', got %q", result)
	}
}

func TestUnmarshalGPUAllocationAnnotation_Empty(t *testing.T) {
	result := UnmarshalGPUAllocationAnnotation([]*GPUAllocation{})
	if result != ";" {
		t.Errorf("expected ';', got %q", result)
	}
}

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	original := []*GPUAllocation{
		{UUID: "uuid1", Vendor: "vendor1", Memory: 1024, Core: 2},
	}
	annotation := UnmarshalGPUAllocationAnnotation(original)
	parsed, err := MarshalGPUAllocationAnnotation(annotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(parsed, original) {
		t.Errorf("round-trip failed: got %+v, want %+v", parsed, original)
	}
}
