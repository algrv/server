package examples

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/algoraveai/server/internal/strudel"
)

// analyzes strudel code and extracts relevant tags
func extractTags(code string, category string, existingTags []string) []string {
	analysis := strudel.AnalyzeCode(code)
	return strudel.GenerateTags(analysis, category, existingTags)
}

// generates a basic description if none exists
func generateDescription(category string, tags []string) string {
	parts := []string{}

	if category != "" {
		parts = append(parts, "A "+strings.ToLower(category)+" pattern")
	} else {
		parts = append(parts, "A Strudel pattern")
	}

	features := []string{}

	for _, tag := range tags {
		switch tag {
		case "drums", "percussion":
			features = append(features, "drums")
		case "melody", "melodic":
			features = append(features, "melody")
		case "bass":
			features = append(features, "bass")
		case "chords", "harmony":
			features = append(features, "chords")
		case "reverb", "delay":
			features = append(features, "effects")
		}
	}

	if len(features) > 0 {
		// deduplicate features
		uniqueFeatures := make(map[string]bool)

		for _, f := range features {
			uniqueFeatures[f] = true
		}

		featureList := make([]string, 0, len(uniqueFeatures))
		for f := range uniqueFeatures {
			featureList = append(featureList, f)
		}

		if len(featureList) == 1 {
			parts = append(parts, "featuring "+featureList[0])
		} else if len(featureList) == 2 {
			parts = append(parts, "featuring "+featureList[0]+" and "+featureList[1])
		} else if len(featureList) > 2 {
			parts = append(parts, "featuring "+strings.Join(featureList[:len(featureList)-1], ", ")+" and "+featureList[len(featureList)-1])
		}
	}

	return strings.Join(parts, " ") + "."
}

func LoadExamplesFromJSON(filePath string) ([]RawExample, error) {
	// read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read examples file: %w", err)
	}

	var rawExamples []RawExample
	if err := json.Unmarshal(data, &rawExamples); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(rawExamples) == 0 {
		return nil, fmt.Errorf("no examples found in file")
	}

	return rawExamples, nil
}
