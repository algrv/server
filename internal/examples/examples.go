package examples

import (
	"fmt"
	"time"
)

// enriches a raw code sample with auto-generated metadata
func ProcessRawExample(raw RawExample) (Example, error) {
	if raw.Title == "" {
		return Example{}, fmt.Errorf("title is required")
	}

	if raw.Code == "" {
		return Example{}, fmt.Errorf("code is required")
	}

	tags := extractTags(raw.Code, raw.Category, raw.Tags)

	description := raw.Description
	if description == "" {
		description = generateDescription(raw.Code, raw.Title, raw.Category, tags)
	}

	if raw.Author == "" {
		raw.Author = "curated"
	}

	return Example{
		Title:       raw.Title,
		Description: description,
		Code:        raw.Code,
		Tags:        tags,
		Author:      raw.Author,
		Category:    raw.Category,
		SourceURL:   "",
		CreatedAt:   time.Now(),
	}, nil
}

func ProcessRawExamples(rawExamples []RawExample) ([]Example, error) {
	examples := make([]Example, 0, len(rawExamples))

	for _, raw := range rawExamples {
		example, err := ProcessRawExample(raw)

		if err != nil {
			return nil, fmt.Errorf("failed to process example: %v", err)
		}

		examples = append(examples, example)
	}

	return examples, nil
}
