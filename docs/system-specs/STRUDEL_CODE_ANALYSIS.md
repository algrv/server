# Strudel Code Analysis Package

## Problem Statement

Both `retriever` and `examples` packages need to parse and analyze Strudel code, but for different purposes:

- **`retriever`**: Extracts keywords from current editor state to enhance contextual search
- **`examples`**: Analyzes code samples to generate semantic tags for categorization

Without standardization, this leads to duplicated regex patterns, inconsistent parsing, and harder maintenance.

## Solution: `internal/strudel` Package

A shared package with three layers of abstraction:

```
Layer 3: Semantic Analysis (analyzer.go)
  → Generate semantic tags, complexity scoring, musical element detection
  → Used by: examples package

Layer 2: Keyword Extraction (keywords.go)
  → Convert parsed elements to search keywords
  → Used by: retriever package

Layer 1: Core Parsing (parser.go)
  → Extract sounds, notes, functions, variables
  → Single source of truth for all regex patterns
  → Used by: Layers 2 & 3
```

## API Design

### Layer 1: Core Parsing (`parser.go`)

```go
type ParsedCode struct {
    Sounds    []string       // ["bd", "hh", "sd"]
    Notes     []string       // ["c", "e", "g"]
    Functions []string       // ["fast", "slow", "stack"]
    Variables []string       // ["pat1", "rhythm"]
    Patterns  map[string]int // {"stack": 2, "arrange": 1}
}

func Parse(code string) ParsedCode
func ExtractSounds(code string) []string
func ExtractNotes(code string) []string
func ExtractFunctions(code string) []string
```

### Layer 2: Keyword Extraction (`keywords.go`)

```go
type KeywordOptions struct {
    MaxKeywords      int
    IncludeSounds    bool
    IncludeNotes     bool
    IncludeFunctions bool
}

func ExtractKeywords(code string) string
func ExtractKeywordsWithOptions(code string, opts KeywordOptions) string
```

### Layer 3: Semantic Analysis (`analyzer.go`)

```go
type CodeAnalysis struct {
    SoundTags      []string // ["drums", "synth", "bass"]
    EffectTags     []string // ["delay", "reverb", "filter"]
    MusicalTags    []string // ["melody", "chords", "rhythm"]
    ComplexityTags []string // ["layered", "advanced", "simple"]
    Complexity     int      // 0-10 score
}

func AnalyzeCode(code string) CodeAnalysis
func GenerateTags(analysis CodeAnalysis, category string, existingTags []string) []string
```

## Implementation Details

### Regex Patterns (Centralized in parser.go)

```go
var (
    soundPattern    = regexp.MustCompile(`(?:sound|s)\s*\(\s*["']([^"']+)["']`)
    notePattern     = regexp.MustCompile(`note\s*\(\s*["']([^"']+)["']`)
    functionPattern = regexp.MustCompile(`\.(\w+)\s*\(`)
    variablePattern = regexp.MustCompile(`(?:let|const|var)\s+(\w+)\s*=`)
)
```

### Semantic Categorization (analyzer.go)

Sound categories map to tags:
- `drums`: bd, hh, sd, cp, oh, ch, cy, rim, clap
- `synth`: sawtooth, sine, square, triangle, piano
- `bass`: bass, subbass

### Complexity Scoring

Score 0-10 based on:
- Code length (>500 chars = +3)
- Stack count (>3 = +3)
- Variable usage (>5 = +2)
- Advanced patterns (arrange = +2)

## Package Usage

### Retriever (before → after)

```go
// Before: 50+ lines of duplicated regex
// After:
func extractEditorKeywords(editorState string) string {
    return strudel.ExtractKeywords(editorState)
}
```

### Examples (before → after)

```go
// Before: 100+ lines of duplicated logic
// After:
func extractTags(code, category string, existing []string) []string {
    analysis := strudel.AnalyzeCode(code)
    return strudel.GenerateTags(analysis, category, existing)
}
```

## Benefits

- Single source of truth for Strudel syntax patterns
- Easy to update when Strudel adds new syntax
- Each layer testable independently
- Reusable for future features (validation, highlighting, auto-complete)
