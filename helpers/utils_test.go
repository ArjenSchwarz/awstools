package helpers

import (
	"os"
	"reflect"
	"testing"
)

func TestStringInSlice(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		slice    []string
		expected bool
	}{
		{
			name:     "string exists in slice",
			target:   "apple",
			slice:    []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "string does not exist in slice",
			target:   "grape",
			slice:    []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "empty slice",
			target:   "apple",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "empty target string",
			target:   "",
			slice:    []string{"apple", "", "cherry"},
			expected: true,
		},
		{
			name:     "case sensitive match",
			target:   "Apple",
			slice:    []string{"apple", "banana", "cherry"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringInSlice(tt.target, tt.slice)
			if result != tt.expected {
				t.Errorf("stringInSlice(%q, %v) = %v, want %v", tt.target, tt.slice, result, tt.expected)
			}
		})
	}
}

func TestGetStringMapFromJSONFile(t *testing.T) {
	// Create a temporary JSON file for testing
	tmpFile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test JSON data
	jsonData := `{"key1": "value1", "key2": "value2", "key3": "value3"}`
	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test the function
	result := GetStringMapFromJSONFile(tmpFile.Name())
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GetStringMapFromJSONFile() = %v, want %v", result, expected)
	}
}

func TestGetStringMapFromJSONFile_InvalidFile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("GetStringMapFromJSONFile should panic on non-existent file")
		}
	}()
	GetStringMapFromJSONFile("non-existent-file.json")
}

func TestGetStringMapFromJSONFile_InvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tmpFile, err := os.CreateTemp("", "test-invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid JSON data
	invalidJSON := `{"key1": "value1", "key2":}`
	if _, err := tmpFile.WriteString(invalidJSON); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("GetStringMapFromJSONFile should panic on invalid JSON")
		}
	}()
	GetStringMapFromJSONFile(tmpFile.Name())
}

func TestFlattenStringMaps(t *testing.T) {
	tests := []struct {
		name     string
		input    []map[string]string
		expected map[string]string
	}{
		{
			name: "single map",
			input: []map[string]string{
				{"key1": "value1", "key2": "value2"},
			},
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "multiple maps no overlap",
			input: []map[string]string{
				{"key1": "value1"},
				{"key2": "value2"},
				{"key3": "value3"},
			},
			expected: map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
		},
		{
			name: "multiple maps with overlap - later values override",
			input: []map[string]string{
				{"key1": "old_value", "key2": "value2"},
				{"key1": "new_value", "key3": "value3"},
			},
			expected: map[string]string{"key1": "new_value", "key2": "value2", "key3": "value3"},
		},
		{
			name:     "empty slice",
			input:    []map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "empty maps",
			input: []map[string]string{
				{},
				{},
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenStringMaps(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FlattenStringMaps() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeByResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "VPC ID",
			resourceID: "vpc-12345678",
			expected:   "vpc",
		},
		{
			name:       "Subnet ID",
			resourceID: "subnet-abcdef12",
			expected:   "subnet",
		},
		{
			name:       "Instance ID",
			resourceID: "i-1234567890abcdef0",
			expected:   "i",
		},
		{
			name:       "Security Group ID",
			resourceID: "sg-12345678",
			expected:   "sg",
		},
		{
			name:       "Route Table ID",
			resourceID: "rtb-12345678",
			expected:   "rtb",
		},
		{
			name:       "Internet Gateway ID",
			resourceID: "igw-12345678",
			expected:   "igw",
		},
		{
			name:       "Transit Gateway ID",
			resourceID: "tgw-12345678",
			expected:   "tgw",
		},
		{
			name:       "Multiple hyphens",
			resourceID: "vpce-svc-12345678",
			expected:   "vpce-svc",
		},
		{
			name:       "Single component",
			resourceID: "resource",
			expected:   "",
		},
		{
			name:       "Empty string",
			resourceID: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TypeByResourceID(tt.resourceID)
			if result != tt.expected {
				t.Errorf("TypeByResourceID(%q) = %q, want %q", tt.resourceID, result, tt.expected)
			}
		})
	}
}
