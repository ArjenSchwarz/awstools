package helpers

import (
	"encoding/json"
	"os"
	"strings"
)

// stringInSlice checks if a string exists in a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// GetStringMapFromJSONFile parses a JSON file and returns it as a string map
func GetStringMapFromJSONFile(filename string) map[string]string {
	originalfile, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	pulledin := make(map[string]string)
	err = json.Unmarshal(originalfile, &pulledin)
	if err != nil {
		panic(err)
	}
	return pulledin
}

// FlattenStringMaps combines multiple stringmaps into a single one.
// Later values will override earlier if duplicates are present
func FlattenStringMaps(stringmaps []map[string]string) map[string]string {
	result := make(map[string]string)
	for _, stringmap := range stringmaps {
		for key, value := range stringmap {
			result[key] = value
		}
	}
	return result
}

// TypeByResourceID identifies the type of resource based on its unique ID
func TypeByResourceID(resourceID string) string {
	portions := strings.Split(resourceID, "-")
	return strings.Join(portions[0:len(portions)-1], "-")
}
