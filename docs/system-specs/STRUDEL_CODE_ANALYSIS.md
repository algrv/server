# Strudel Code Analysis Package

## Problem Statement

Both `retriever` and `examples` packages need to parse and analyze Strudel code, but for different purposes:

- **`retriever`**: Extracts keywords from current editor state to enhance contextual search
- **`examples`**: Analyzes code samples to generate semantic tags for categorization

Without standardization, this leads to:
- ❌ Duplicated regex patterns
- ❌ Inconsistent parsing logic
- ❌ Harder to maintain when Strudel syntax changes
- ❌ No single source of truth

## Solution: `internal/strudel` Package

A shared package with three layers of abstraction:

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Semantic Analysis (analyzer.go)                   │
│ ─────────────────────────────────────────────────────────── │
│ • Generate semantic tags (drums, melody, complex)           │
│ • Complexity scoring                                        │
│ • Musical element detection                                 │
│ • Used by: examples package                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Keyword Extraction (keywords.go)                   │
│ ─────────────────────────────────────────────────────────── │
│ • Convert parsed elements to search keywords                │
│ • Deduplication and filtering                               │
│ • Limit keyword count                                       │
│ • Used by: retriever package                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Core Parsing (parser.go)                           │
│ ─────────────────────────────────────────────────────────── │
│ • Extract sounds: sound("bd") → ["bd"]                      │
│ • Extract notes: note("c e g") → ["c", "e", "g"]            │
│ • Extract functions: .fast(2) → ["fast"]                    │
│ • Count patterns: stack, variables, etc.                    │
│ • Single source of truth for all regex patterns             │
│ • Used by: Layers 2 & 3                                     │
└─────────────────────────────────────────────────────────────┘
```

## API Design

### Layer 1: Core Parsing (`parser.go`)

Low-level extraction functions that return raw data:

```go
package strudel

// ParsedCode contains all extracted elements
type ParsedCode struct {
    Sounds      []string          // ["bd", "hh", "sd"]
    Notes       []string          // ["c", "e", "g"]
    Functions   []string          // ["fast", "slow", "stack"]
    Variables   []string          // ["pat1", "rhythm"]
    Patterns    map[string]int    // {"stack": 2, "arrange": 1}
}

// Parse extracts all elements from Strudel code
func Parse(code string) ParsedCode

// Individual extractors (used internally by Parse)
func ExtractSounds(code string) []string
func ExtractNotes(code string) []string
func ExtractFunctions(code string) []string
func ExtractVariables(code string) []string
func CountPattern(code string, pattern string) int
```

**Example:**
```go
code := `sound("bd").fast(2).stack(note("c e g"))`
parsed := strudel.Parse(code)

// parsed.Sounds = ["bd"]
// parsed.Notes = ["c", "e", "g"]
// parsed.Functions = ["fast", "stack"]
// parsed.Patterns = {"stack": 1}
```

### Layer 2: Keyword Extraction (`keywords.go`)

Converts parsed code into search-optimized keywords:

```go
package strudel

type KeywordOptions struct {
    MaxKeywords      int  // Limit total keywords (default: 10)
    IncludeSounds    bool // Include sound names
    IncludeNotes     bool // Include note names
    IncludeFunctions bool // Include function names
    Deduplicate      bool // Remove duplicates (default: true)
}

// ExtractKeywords extracts keywords with default options
// Returns: space-separated keyword string for search
func ExtractKeywords(code string) string

// ExtractKeywordsWithOptions allows custom options
func ExtractKeywordsWithOptions(code string, opts KeywordOptions) string
```

**Example (Retriever Use Case):**
```go
editorState := `sound("bd hh sd").fast(4).stack(note("c a f e"))`

// Extract keywords for contextual search
keywords := strudel.ExtractKeywords(editorState)
// Returns: "bd hh sd fast stack c a f e"
// (limited to 10 keywords, deduplicated)

// With custom options
keywords := strudel.ExtractKeywordsWithOptions(editorState, strudel.KeywordOptions{
    MaxKeywords: 5,
    IncludeSounds: true,
    IncludeNotes: false,
    IncludeFunctions: true,
})
// Returns: "bd hh sd fast stack"
```

### Layer 3: Semantic Analysis (`analyzer.go`)

Generates semantic tags and complexity scores:

```go
package strudel

type CodeAnalysis struct {
    // Categorized tags
    SoundTags      []string // ["drums", "synth", "bass"]
    EffectTags     []string // ["delay", "reverb", "filter"]
    MusicalTags    []string // ["melody", "chords", "rhythm"]
    ComplexityTags []string // ["layered", "advanced", "simple"]

    // Metrics
    Complexity     int      // 0-10 score
    LineCount      int
    FunctionCount  int
    VariableCount  int
}

// AnalyzeCode performs full semantic analysis
func AnalyzeCode(code string) CodeAnalysis

// GenerateTags combines analysis with existing metadata
func GenerateTags(analysis CodeAnalysis, category string, existingTags []string) []string
```

**Example (Examples Package Use Case):**
```go
code := `
sound("bd").fast(4)
  .stack(sound("hh").fast(8))
  .stack(note("c e g").sound("sawtooth").delay(0.25))
`

analysis := strudel.AnalyzeCode(code)

// analysis.SoundTags = ["drums", "percussion", "synth"]
// analysis.EffectTags = ["delay"]
// analysis.MusicalTags = ["melody", "chords", "layered"]
// analysis.ComplexityTags = ["intermediate", "layered"]
// analysis.Complexity = 6

// Generate final tags for database
tags := strudel.GenerateTags(analysis, "techno", []string{"tutorial"})
// Returns: ["techno", "tutorial", "drums", "percussion", "synth", "delay",
//           "melody", "chords", "layered", "intermediate"]
```

## Implementation Details

### Regex Patterns (Centralized)

All patterns defined once in `parser.go`:

```go
var (
    // Sound extraction: sound("bd") or s("bd")
    soundPattern = regexp.MustCompile(`(?:sound|s)\s*\(\s*["']([^"']+)["']`)

    // Note extraction: note("c e g")
    notePattern = regexp.MustCompile(`note\s*\(\s*["']([^"']+)["']`)

    // Function calls: .fast(2), .slow(4)
    functionPattern = regexp.MustCompile(`\.(\w+)\s*\(`)

    // Variable declarations: let x = ..., const y = ...
    variablePattern = regexp.MustCompile(`(?:let|const|var)\s+(\w+)\s*=`)

    // Scale/mode: scale("minor"), mode("dorian")
    scalePattern = regexp.MustCompile(`(?:scale|mode)\s*\(\s*["'](\w+)["']`)
)
```

### Semantic Categorization Logic

Defined in `analyzer.go`:

```go
// Sound categorization
var soundCategories = map[string][]string{
    "drums": {"bd", "hh", "sd", "cp", "oh", "ch", "cy", "rim", "clap"},
    "synth": {"sawtooth", "sine", "square", "triangle", "piano"},
    "bass":  {"bass", "subbass"},
}

// Effect categorization
var effectCategories = map[string]string{
    "delay":      "delay",
    "room":       "reverb",
    "lpf":        "filter",
    "hpf":        "filter",
    "crush":      "distortion",
    "gain":       "dynamics",
    "pan":        "spatial",
}

// Musical element detection
func detectMusicalElements(parsed ParsedCode) []string {
    tags := []string{}

    if len(parsed.Notes) > 0 {
        tags = append(tags, "melody", "melodic")
    }

    if countPattern(parsed.Functions, "scale") > 0 {
        tags = append(tags, "scales", "melodic")
    }

    // Chord detection (multiple notes)
    if hasChords(parsed.Notes) {
        tags = append(tags, "chords", "harmony")
    }

    return tags
}
```

### Complexity Scoring Algorithm

```go
func calculateComplexity(parsed ParsedCode) int {
    score := 0

    // Base complexity from code length
    if len(parsed.Code) > 500 {
        score += 3
    } else if len(parsed.Code) > 200 {
        score += 2
    } else {
        score += 1
    }

    // Layering complexity
    stackCount := parsed.Patterns["stack"]
    if stackCount > 3 {
        score += 3
    } else if stackCount > 0 {
        score += 2
    }

    // Variable usage
    if len(parsed.Variables) > 5 {
        score += 2
    } else if len(parsed.Variables) > 0 {
        score += 1
    }

    // Advanced patterns
    if parsed.Patterns["arrange"] > 0 {
        score += 2
    }

    // Interactive elements
    if parsed.Patterns["slider"] > 0 {
        score += 1
    }

    // Cap at 10
    if score > 10 {
        score = 10
    }

    return score
}
```

## Package Usage Examples

### Retriever Package

**Before (duplicated logic):**
```go
// retriever/utils.go
func extractEditorKeywords(editorState string) string {
    keywords := []string{}

    // Duplicated regex patterns
    soundRegex := regexp.MustCompile(`sound\("(\w+)"\)`)
    for _, match := range soundRegex.FindAllStringSubmatch(editorState, -1) {
        keywords = append(keywords, match[1])
    }

    // More duplicated patterns...
    // ... 50+ lines of code ...

    return strings.Join(keywords, " ")
}
```

**After (using strudel package):**
```go
// retriever/utils.go
import "algorave/internal/strudel"

func extractEditorKeywords(editorState string) string {
    return strudel.ExtractKeywords(editorState)
}
```

### Examples Package

**Before (duplicated logic):**
```go
// examples/utils.go
func extractTags(code string, category string, existingTags []string) []string {
    tags := make(map[string]bool)

    // Duplicated sound extraction
    soundRegex := regexp.MustCompile(`sound\("(\w+)"\)`)
    // ... 100+ lines of duplicated categorization logic ...

    // Duplicated complexity analysis
    stackCount := len(regexp.MustCompile(`stack\s*\(`).FindAllString(code, -1))
    // ... 50+ lines of complexity logic ...

    return tagSlice
}
```

**After (using strudel package):**
```go
// examples/utils.go
import "algorave/internal/strudel"

func extractTags(code string, category string, existingTags []string) []string {
    analysis := strudel.AnalyzeCode(code)
    return strudel.GenerateTags(analysis, category, existingTags)
}
```

## Testing Strategy

### Layer 1: Core Parsing Tests
```go
func TestExtractSounds(t *testing.T) {
    tests := []struct {
        code     string
        expected []string
    }{
        {`sound("bd")`, []string{"bd"}},
        {`sound("bd hh sd")`, []string{"bd", "hh", "sd"}},
        {`s("bd").stack(s("hh"))`, []string{"bd", "hh"}},
    }

    for _, tt := range tests {
        result := strudel.ExtractSounds(tt.code)
        assert.Equal(t, tt.expected, result)
    }
}
```

### Layer 2: Keyword Extraction Tests
```go
func TestExtractKeywords(t *testing.T) {
    code := `sound("bd").fast(2).stack(note("c e g"))`
    keywords := strudel.ExtractKeywords(code)

    assert.Contains(t, keywords, "bd")
    assert.Contains(t, keywords, "fast")
    assert.Contains(t, keywords, "c")
}
```

### Layer 3: Semantic Analysis Tests
```go
func TestAnalyzeCode_Drums(t *testing.T) {
    code := `sound("bd hh sd").fast(4)`
    analysis := strudel.AnalyzeCode(code)

    assert.Contains(t, analysis.SoundTags, "drums")
    assert.Contains(t, analysis.SoundTags, "percussion")
    assert.GreaterOrEqual(t, analysis.Complexity, 3)
}
```

## Benefits

### 1. Single Source of Truth
- ✅ All Strudel syntax patterns defined once
- ✅ Easy to update when Strudel adds new syntax
- ✅ Consistent parsing across entire codebase

### 2. Better Maintainability
- ✅ Changes to parsing logic require updates in one place
- ✅ Clear separation of concerns (parsing vs analysis vs keywords)
- ✅ Easier to understand and modify

### 3. Testability
- ✅ Each layer can be tested independently
- ✅ Mock parsed data for higher layers
- ✅ Comprehensive test coverage for all Strudel patterns

### 4. Reusability
- ✅ Can be used by future features:
  - Code validation
  - Syntax highlighting
  - Auto-completion
  - Code complexity metrics dashboard
  - Pattern detection

### 5. Performance
- ✅ Compile regex patterns once (not per call)
- ✅ Efficient parsing with single pass
- ✅ Option to cache parsed results if needed

## Migration Plan

### Phase 1: Create Package
1. Create `internal/strudel/` directory
2. Implement `parser.go` with core extraction
3. Write comprehensive tests for parser
4. Implement `keywords.go` for retriever use case
5. Implement `analyzer.go` for examples use case

### Phase 2: Migrate Retriever
1. Update `retriever/utils.go` to use `strudel.ExtractKeywords()`
2. Remove duplicated extraction logic
3. Run retriever tests to ensure compatibility
4. Verify search results quality

### Phase 3: Migrate Examples
1. Update `examples/utils.go` to use `strudel.AnalyzeCode()`
2. Remove duplicated categorization logic
3. Run examples tests to ensure tag quality
4. Verify tag generation accuracy

### Phase 4: Cleanup
1. Remove old regex patterns from retriever and examples
2. Update documentation
3. Add integration tests
4. Performance benchmarking

## Future Enhancements

### Pattern Library
Store common Strudel patterns for detection:
```go
var commonPatterns = map[string]string{
    "four-on-floor": `sound\("bd"\)\.fast\(4\)`,
    "arpeggio":      `note\([^)]+\)\.every\([^)]+\)`,
}
```

### Code Quality Metrics
```go
type QualityMetrics struct {
    Readability    int // 0-10
    Performance    int // 0-10
    Creativity     int // 0-10
    BestPractices  []string
    Warnings       []string
}

func AnalyzeQuality(code string) QualityMetrics
```

### Auto-completion Support
```go
type CompletionSuggestion struct {
    Text        string
    Type        string // "function", "sound", "note"
    Description string
    Example     string
}

func GetCompletions(code string, cursorPos int) []CompletionSuggestion
```

## Conclusion

The `internal/strudel` package provides a clean, maintainable, and extensible solution for Strudel code analysis. By centralizing parsing logic and providing clear layers of abstraction, we ensure consistency and make future enhancements easier to implement.
