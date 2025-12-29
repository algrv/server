package strudel

import (
	"regexp"
	"strings"
)

var (
	// sound extraction: sound("bd") or s("bd")
	// supports both quotes and backticks: s("bd") or s(`bd`)
	soundPattern = regexp.MustCompile("(?:sound|s)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")

	// note extraction: note("c e g")
	// supports both quotes and backticks: note("c e g") or note(`c e g`)
	notePattern = regexp.MustCompile("note\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")

	// backtick template literals followed by methods: `c e g`.note()
	// extracts content from backticks when followed by a method call
	backtickPattern = regexp.MustCompile("`([^`]+)`\\s*\\.\\w+\\s*\\(")

	// function calls: .fast(2), .slow(4), .stack()
	functionPattern = regexp.MustCompile(`\.(\w+)\s*\(`)

	// variable declarations: let x = ..., const y = ...
	variablePattern = regexp.MustCompile(`(?:let|const|var)\s+(\w+)\s*=`)

	// scale/mode: scale("minor"), mode("dorian")
	scalePattern = regexp.MustCompile(`(?:scale|mode)\s*\(\s*["'](\w+)["']`)
)

// extracts all elements from Strudel code
func Parse(code string) ParsedCode {
	return ParsedCode{
		Sounds:    ExtractSounds(code),
		Notes:     ExtractNotes(code),
		Functions: ExtractFunctions(code),
		Variables: ExtractVariables(code),
		Scales:    ExtractScales(code),
		Patterns:  CountPatterns(code),
	}
}

// extracts sound sample names from sound() calls
// example: sound("bd hh sd") → ["bd", "hh", "sd"]
// handles complex patterns: s("bd:0") → ["bd"], s("[~ sd:3]*2") → ["sd"]
func ExtractSounds(code string) []string {
	sounds := []string{}
	seen := make(map[string]bool)

	matches := soundPattern.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// parse pattern string to extract sound names
			parsed := parsePatternString(match[1])
			for _, s := range parsed {
				if !seen[s] {
					sounds = append(sounds, s)
					seen[s] = true
				}
			}
		}
	}

	return sounds
}

// parsePatternString extracts sound names from Strudel pattern syntax
// handles: "bd:0" → ["bd"], "[~ sd hh]*2" → ["sd", "hh"], "bd, hh" → ["bd", "hh"]
func parsePatternString(pattern string) []string {
	// remove common pattern syntax characters
	cleaners := []string{"[", "]", "<", ">", "(", ")", "{", "}", "*", "@", "!", "/", "|", "?"}
	cleaned := pattern
	for _, char := range cleaners {
		cleaned = strings.ReplaceAll(cleaned, char, " ")
	}

	// split on spaces and commas
	cleaned = strings.ReplaceAll(cleaned, ",", " ")
	tokens := strings.Fields(cleaned)

	sounds := []string{}
	for _, token := range tokens {
		// skip rests and silences
		if token == "~" || token == "-" || token == "" {
			continue
		}

		// remove sample number suffix (e.g., "bd:0" → "bd")
		parts := strings.Split(token, ":")
		soundName := parts[0]

		// skip empty, numeric-only tokens, and common pattern markers
		if soundName == "" || isNumeric(soundName) || soundName == "x" {
			continue
		}

		sounds = append(sounds, soundName)
	}

	return sounds
}

// isNumeric checks if a string is purely numeric
func isNumeric(s string) bool {
	matched, _ := regexp.MatchString(`^\d+\.?\d*$`, s) //nolint:errcheck
	return matched
}

// extractNotes extracts note names from note() calls
// example: note("c e g") → ["c", "e", "g"]
// also supports backtick strings: `c e g`.note() → ["c", "e", "g"]
func ExtractNotes(code string) []string {
	notes := []string{}

	// extract from note("...") or note(`...`)
	matches := notePattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// split space-separated notes
			noteList := strings.Fields(match[1])
			notes = append(notes, noteList...)
		}
	}

	// extract from backtick template literals: `c e g`.note()
	backtickMatches := backtickPattern.FindAllStringSubmatch(code, -1)
	for _, match := range backtickMatches {
		if len(match) > 1 {
			// split space-separated notes/patterns
			noteList := strings.Fields(match[1])
			notes = append(notes, noteList...)
		}
	}

	return notes
}

// extractFunctions extracts function/method names from .func() calls
// example: .fast(2).slow(4) → ["fast", "slow"]
func ExtractFunctions(code string) []string {
	functions := []string{}
	matches := functionPattern.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 1 {
			functions = append(functions, match[1])
		}
	}

	return functions
}

// extractVariables extracts variable names from declarations
// example: let pat1 = sound("bd") → ["pat1"]
func ExtractVariables(code string) []string {
	variables := []string{}
	matches := variablePattern.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 1 {
			variables = append(variables, match[1])
		}
	}

	return variables
}

// extractScales extracts scale/mode names
// example: scale("minor") → ["minor"]
func ExtractScales(code string) []string {
	scales := []string{}
	matches := scalePattern.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 1 {
			scales = append(scales, match[1])
		}
	}

	return scales
}

// counts occurrences of specific patterns
func CountPatterns(code string) map[string]int {
	patterns := make(map[string]int)

	// common patterns to count (extracted from Strudel docs)
	patternRegexes := map[string]*regexp.Regexp{
		// structure/layering (match both .stack() and standalone stack())
		"stack": regexp.MustCompile(`(?:^|\W)stack\s*\(`),
		"layer": regexp.MustCompile(`(?:^|\W)layer\s*\(`),

		// time modifiers
		"slow":    regexp.MustCompile(`\.slow\s*\(`),
		"fast":    regexp.MustCompile(`\.fast\s*\(`),
		"early":   regexp.MustCompile(`\.early\s*\(`),
		"late":    regexp.MustCompile(`\.late\s*\(`),
		"euclid":  regexp.MustCompile(`\.euclid\s*\(`),
		"rev":     regexp.MustCompile(`\.rev\s*\(`),
		"iter":    regexp.MustCompile(`\.iter\s*\(`),
		"ply":     regexp.MustCompile(`\.ply\s*\(`),
		"segment": regexp.MustCompile(`\.segment\s*\(`),

		// conditional modifiers
		"every":        regexp.MustCompile(`\.every\s*\(`),
		"sometimes":    regexp.MustCompile(`\.sometimes\s*\(`),
		"often":        regexp.MustCompile(`\.often\s*\(`),
		"rarely":       regexp.MustCompile(`\.rarely\s*\(`),
		"almostNever":  regexp.MustCompile(`\.almostNever\s*\(`),
		"almostAlways": regexp.MustCompile(`\.almostAlways\s*\(`),
		"never":        regexp.MustCompile(`\.never\s*\(`),
		"always":       regexp.MustCompile(`\.always\s*\(`),

		// arrangement
		"arrange": regexp.MustCompile(`(?:^|\W)arrange\s*\(`), // match arrange() as function or method

		// interactive
		"slider": regexp.MustCompile(`slider\s*\(`),

		// sampler effects
		"chop":    regexp.MustCompile(`\.chop\s*\(`),
		"striate": regexp.MustCompile(`\.striate\s*\(`),
		"slice":   regexp.MustCompile(`\.slice\s*\(`),
	}

	for name, regex := range patternRegexes {
		matches := regex.FindAllString(code, -1)
		patterns[name] = len(matches)
	}

	return patterns
}

// counts occurrences of a specific pattern
func CountPattern(code string, pattern string) int {
	regex := regexp.MustCompile(`\.` + pattern + `\s*\(`)
	matches := regex.FindAllString(code, -1)

	return len(matches)
}
