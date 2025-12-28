# Code Annotation System Specification

Documentation for expanding the LLM package to implement annotations for surgical precision on final outputs.

---

## 🎯 Implementation Status

### Phase 1: Enhanced Instructions - ✅ COMPLETED & VALIDATED

**Implemented:** 2025-12-28
**Testing:** Manual testing completed
**Result:** **SUCCESS - Surgical accuracy achieved**

#### What Was Implemented

Enhanced the system prompt instructions in `/internal/agent/prompt.go` with:

1. **Request Type Classification** - LLM now categorizes requests into 4 types:
   - A. ADDITIVE - "add", "create" → append new code
   - B. MODIFICATION - "make", "change" → modify specific element only
   - C. DELETION - "remove", "delete" → remove specific element only
   - D. QUESTIONS - "how", "what" → provide explanation

2. **4-Step Surgical Precision Process** for modifications/deletions:
   - Step 1: IDENTIFY the target element
   - Step 2: LOCATE the exact line in editor state
   - Step 3: MAKE THE CHANGE surgically
   - Step 4: PRESERVE everything else

3. **5 Detailed Few-Shot Examples** showing expected behavior for each request type

#### Test Results (2025-12-28)

**Manual Testing Summary:**
- ✅ ADDITIVE requests: New code appended, existing code preserved exactly
- ✅ MODIFICATION requests: Only target element changed, rest untouched
- ✅ DELETION requests: Only target removed, rest preserved
- ✅ COMPLEX scenarios: Multi-step modifications with perfect preservation

**Notable Test Case - Hi-Hat Stack Behavior:**

Sequence tested:
1. Add drums in a stack
2. Add bass
3. Remove hi-hat from drum stack
4. Add new hi-hats from a different drum kit

**Agent behavior:** Created new hi-hat pattern as separate line, NOT added back to original stack

**Why this is CORRECT:**
- Request was "add new hihats" (ADDITIVE), not "add hihats to the stack"
- Agent made no assumptions about code structure
- Preserved existing stack structure exactly
- User retains full control over organization

**Conclusion:** Phase 1 provides sufficient precision. No need for Phase 2 annotation system at this time.

---

### Phase 2: Code Annotation System - ⏸️ ON HOLD

**Status:** Not currently needed

Phase 1's enhanced instructions achieve the required precision without:
- Additional API calls (+10% cost)
- Added latency (extra LLM round-trip)
- Implementation complexity (new interfaces, confidence scoring, etc.)

**When to implement Phase 2:**
- If precision issues emerge with edge cases over time
- If users request preview of changes before generation
- If metrics show frequent unwanted modifications

The full specification below remains as reference for future implementation if needed.

---

## Overview

The Code Annotation System enables precise, iterative code editing by using a lightweight LLM to analyze user requests and annotate specific code blocks that should be modified, deleted, or added. This improves the main generator's ability to make surgical changes without accidentally altering code the user wants preserved.

**Problem Being Solved:**
- Users request modifications ("make the bass quieter") but the main LLM must infer which specific line to change
- Risk of accidentally modifying code the user wants untouched
- Difficulty making precise changes to complex multi-pattern editor states

**Solution:**
A two-stage pipeline where a small, fast LLM (Haiku) annotates the code with explicit directives before the main LLM (Sonnet) generates the final code.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Integration with LLM Interface](#integration-with-llm-interface)
3. [Annotation Directives](#annotation-directives)
4. [Annotator Interface Design](#annotator-interface-design)
5. [Implementation Details](#implementation-details)
6. [System Prompts](#system-prompts)
7. [Decision Logic: When to Annotate](#decision-logic-when-to-annotate)
8. [Confidence Scoring & Fallback](#confidence-scoring--fallback)
9. [Configuration](#configuration)
10. [Implementation Phases](#implementation-phases)
11. [Testing Strategy](#testing-strategy)
12. [Metrics & Monitoring](#metrics--monitoring)

---

## Architecture Overview

### Current Flow (Phase 1)
```
User Query
    ↓
AnalyzeQuery (Haiku) → Check actionability
    ↓
HybridSearch → Retrieve docs + examples
    ↓
buildSystemPrompt → Construct context
    ↓
GenerateText (Sonnet) → Generate complete code
    ↓
Response
```

### Enhanced Flow (Phase 2 - With Annotation)
```
User Query
    ↓
AnalyzeQuery (Haiku) → Check actionability
    ↓
RequestClassifier → Determine if annotation needed
    ├─ NEW/ADD → Skip annotation
    └─ MODIFY/DELETE → Use annotation
        ↓
    AnnotateCode (Haiku) → Mark specific lines
        ↓
    [Confidence Check]
        ├─ Low confidence → Skip annotation (fallback)
        └─ High confidence → Use annotated code
            ↓
HybridSearch → Retrieve docs + examples
    ↓
buildSystemPrompt → Construct context (with annotations if present)
    ↓
GenerateText (Sonnet) → Generate complete code
    ↓
Response
```

### Benefits

✅ **Precision** - Changes only what user asks for, preserves everything else
✅ **Preservation** - Explicit KEEP AS-IS prevents accidental modifications
✅ **User Experience** - Iterative refinement feels natural and predictable
✅ **Cost Efficient** - Annotator is small/cheap (Haiku: ~$0.0003 vs Sonnet: ~$0.003)
✅ **Debuggable** - Can show user what will change before generation
✅ **Confidence-aware** - Falls back gracefully when uncertain

### Costs

Without Annotation:
- 1 request = 1 Sonnet call (~$0.003)

With Annotation:
- 1 request = 1 Haiku call (~$0.0003) + 1 Sonnet call (~$0.003) = **~$0.0033**
- **+10% cost increase** for significantly better precision

---

## Integration with LLM Interface

### Current LLM Interface (`internal/llm/types.go`)

```go
type LLM interface {
    QueryTransformer
    Embedder
    TextGenerator
}
```

### Enhanced LLM Interface (Phase 2)

```go
// Annotator adds code annotation capabilities
type Annotator interface {
    AnnotateCode(ctx context.Context, req AnnotationRequest) (*AnnotationResult, error)
}

// AnnotationRequest contains inputs for code annotation
type AnnotationRequest struct {
    UserQuery       string    // the modification request
    CurrentCode     string    // existing editor state
    ConversationHistory []Message // for context
}

// AnnotationResult contains the annotated code and metadata
type AnnotationResult struct {
    AnnotatedCode   string              // code with directive comments
    Annotations     []Annotation        // structured list of annotations
    Confidence      float64             // 0.0 - 1.0 confidence score
    ShouldRewrite   bool                // true if complete rewrite needed
    UseFallback     bool                // true if annotation failed/low confidence
}

// Annotation describes a single directive
type Annotation struct {
    LineNumber  int       // 1-indexed line number
    Directive   Directive // the type of annotation
    Reason      string    // explanation of why
    Target      string    // what's being modified (e.g., "kick drum pattern")
}

// Directive types
type Directive string

const (
    DirectiveModify  Directive = "MODIFY THIS"
    DirectiveDelete  Directive = "DELETE THIS"
    DirectiveAdd     Directive = "ADD NEW CODE BELOW"
    DirectiveKeep    Directive = "KEEP AS-IS"
    DirectiveRewrite Directive = "REWRITE ENTIRE PATTERN"
)

// Updated LLM interface with annotation support
type LLM interface {
    QueryTransformer
    Embedder
    TextGenerator
    Annotator
}
```

### Configuration Extension

```go
// internal/llm/types.go - Config struct
type Config struct {
    // ... existing fields ...

    // annotator configuration (code annotation)
    AnnotatorProvider    Provider
    AnnotatorAPIKey      string
    AnnotatorModel       string  // e.g., "claude-3-haiku-20240307"
    AnnotatorMaxTokens   int     // e.g., 500
    AnnotatorTemperature float32 // e.g., 0.2 (low for consistency)

    // annotation behavior
    EnableAnnotation           bool    // toggle feature on/off
    AnnotationConfidenceThreshold float64 // minimum confidence to use (e.g., 0.7)
}
```

---

## Annotation Directives

### Directive Syntax

```javascript
// MODIFY THIS: [reason]
// - Change this specific line/block

// DELETE THIS: [reason]
// - Remove this line/block entirely

// ADD NEW CODE BELOW: [reason]
// - Insert new code after this marker

// KEEP AS-IS
// - Don't change this (for clarity)

// REWRITE ENTIRE PATTERN: [reason]
// - Complete restructure needed
```

### Examples

#### Example 1: Modify Specific Element

**User:** "make the kick quieter"

**Current Code:**
```javascript
sound("bd*4").bank("RolandTR909")
$: sound("hh*8")
```

**Annotated Code:**
```javascript
// MODIFY THIS: Reduce kick volume by adding .gain()
sound("bd*4").bank("RolandTR909")
$: sound("hh*8")
```

**Generated Code:**
```javascript
sound("bd*4").bank("RolandTR909").gain(0.6)
$: sound("hh*8")
```

#### Example 2: Delete Specific Element

**User:** "remove the hi-hats"

**Current Code:**
```javascript
$: sound("bd*4")
$: sound("hh*8")
$: note("c2 e2 g2").sound("sawtooth")
```

**Annotated Code:**
```javascript
$: sound("bd*4")
// DELETE THIS: Remove hi-hat pattern as requested
$: sound("hh*8")
$: note("c2 e2 g2").sound("sawtooth")
```

**Generated Code:**
```javascript
$: sound("bd*4")
$: note("c2 e2 g2").sound("sawtooth")
```

#### Example 3: Add New Element

**User:** "add a melody"

**Current Code:**
```javascript
$: sound("bd*4")
$: sound("hh*8")
```

**Annotated Code:**
```javascript
$: sound("bd*4")
$: sound("hh*8")
// ADD NEW CODE BELOW: Add a melodic pattern that complements the drums
```

**Generated Code:**
```javascript
$: sound("bd*4")
$: sound("hh*8")
$: note("c3 e3 g3 c4").sound("piano").slow(2)
```

#### Example 4: Multiple Modifications

**User:** "make it more minimal - remove melody, simplify drums"

**Current Code:**
```javascript
$: sound("bd sd bd sd").fast(4)
$: sound("hh:1 hh:2 hh:3").fast(16)
$: note("c e g c5 e5").sound("piano")
```

**Annotated Code:**
```javascript
// MODIFY THIS: Simplify to just steady kick pattern
$: sound("bd sd bd sd").fast(4)
// DELETE THIS: Remove complex hi-hat pattern
$: sound("hh:1 hh:2 hh:3").fast(16)
// DELETE THIS: Remove melody as requested
$: note("c e g c5 e5").sound("piano")
```

**Generated Code:**
```javascript
$: sound("bd*4")
```

---

## Annotator Interface Design

### Implementation in `internal/llm/anthropic.go`

```go
// AnnotateCode implements the Annotator interface
func (a *AnthropicLLM) AnnotateCode(ctx context.Context, req AnnotationRequest) (*AnnotationResult, error) {
    // Build annotator prompt
    prompt := buildAnnotatorPrompt(req)

    // Call Haiku model
    messages := []anthropic.Message{
        {
            Role: anthropic.RoleUser,
            Content: []anthropic.ContentBlock{
                anthropic.NewTextBlock(prompt),
            },
        },
    }

    response, err := a.annotatorClient.Messages.New(ctx, anthropic.MessageNewParams{
        Model:       anthropic.F(a.annotatorModel),
        MaxTokens:   anthropic.Int(a.annotatorMaxTokens),
        Temperature: anthropic.Float(a.annotatorTemperature),
        Messages:    anthropic.F(messages),
    })

    if err != nil {
        return nil, fmt.Errorf("failed to annotate code: %w", err)
    }

    // Parse response
    result := parseAnnotationResponse(response.Content[0].Text, req.CurrentCode)

    return result, nil
}

// buildAnnotatorPrompt constructs the prompt for code annotation
func buildAnnotatorPrompt(req AnnotationRequest) string {
    var builder strings.Builder

    builder.WriteString(annotatorSystemPrompt)
    builder.WriteString("\n\n")

    // Add conversation history if present
    if len(req.ConversationHistory) > 0 {
        builder.WriteString("Conversation history:\n")
        for i, msg := range req.ConversationHistory {
            builder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, msg.Role, msg.Content))
        }
        builder.WriteString("\n")
    }

    builder.WriteString(fmt.Sprintf("User request: %s\n\n", req.UserQuery))
    builder.WriteString("Current code:\n```\n")
    builder.WriteString(req.CurrentCode)
    builder.WriteString("\n```\n\n")
    builder.WriteString("Output annotated code:")

    return builder.String()
}

// parseAnnotationResponse extracts structured annotations from LLM response
func parseAnnotationResponse(response string, originalCode string) *AnnotationResult {
    // Extract annotated code from markdown code block if present
    annotatedCode := extractCodeBlock(response)
    if annotatedCode == "" {
        annotatedCode = response // fallback to raw response
    }

    // Parse annotations from comments
    annotations := []Annotation{}
    lines := strings.Split(annotatedCode, "\n")

    for i, line := range lines {
        trimmed := strings.TrimSpace(line)

        // Check for directives
        if strings.HasPrefix(trimmed, "// MODIFY THIS:") {
            annotations = append(annotations, Annotation{
                LineNumber: i + 1,
                Directive:  DirectiveModify,
                Reason:     extractReason(trimmed, "// MODIFY THIS:"),
                Target:     extractTarget(lines, i),
            })
        } else if strings.HasPrefix(trimmed, "// DELETE THIS:") {
            annotations = append(annotations, Annotation{
                LineNumber: i + 1,
                Directive:  DirectiveDelete,
                Reason:     extractReason(trimmed, "// DELETE THIS:"),
                Target:     extractTarget(lines, i),
            })
        } else if strings.HasPrefix(trimmed, "// ADD NEW CODE BELOW:") {
            annotations = append(annotations, Annotation{
                LineNumber: i + 1,
                Directive:  DirectiveAdd,
                Reason:     extractReason(trimmed, "// ADD NEW CODE BELOW:"),
            })
        } else if strings.HasPrefix(trimmed, "// REWRITE ENTIRE PATTERN:") {
            annotations = append(annotations, Annotation{
                LineNumber: i + 1,
                Directive:  DirectiveRewrite,
                Reason:     extractReason(trimmed, "// REWRITE ENTIRE PATTERN:"),
            })
        }
    }

    // Calculate confidence based on:
    // - Number of annotations (too many = uncertain)
    // - Specificity of reasons
    // - Whether annotations align with user query
    confidence := calculateConfidence(annotations, len(lines))

    // Check if complete rewrite requested
    shouldRewrite := hasDirective(annotations, DirectiveRewrite)

    // Determine if we should fall back to non-annotated flow
    useFallback := confidence < 0.7 || len(annotations) == 0

    return &AnnotationResult{
        AnnotatedCode: annotatedCode,
        Annotations:   annotations,
        Confidence:    confidence,
        ShouldRewrite: shouldRewrite,
        UseFallback:   useFallback,
    }
}

// calculateConfidence scores the annotation quality
func calculateConfidence(annotations []Annotation, totalLines int) float64 {
    if len(annotations) == 0 {
        return 0.0
    }

    baseConfidence := 0.9

    // Reduce confidence if too many annotations (indicates uncertainty)
    if len(annotations) > 3 {
        baseConfidence -= 0.1 * float64(len(annotations)-3)
    }

    // Reduce confidence if reasons are vague
    for _, ann := range annotations {
        if len(ann.Reason) < 10 {
            baseConfidence -= 0.1
        }
    }

    // Ensure confidence is in [0.0, 1.0]
    if baseConfidence < 0.0 {
        baseConfidence = 0.0
    }
    if baseConfidence > 1.0 {
        baseConfidence = 1.0
    }

    return baseConfidence
}

// Helper functions
func extractReason(line string, prefix string) string {
    reason := strings.TrimPrefix(line, prefix)
    return strings.TrimSpace(reason)
}

func extractTarget(lines []string, commentLineIndex int) string {
    // Look at next non-empty line to identify target
    for i := commentLineIndex + 1; i < len(lines); i++ {
        line := strings.TrimSpace(lines[i])
        if line != "" && !strings.HasPrefix(line, "//") {
            return line
        }
    }
    return ""
}

func extractCodeBlock(text string) string {
    // Remove markdown code blocks if present
    code := strings.TrimPrefix(text, "```javascript")
    code = strings.TrimPrefix(code, "```js")
    code = strings.TrimPrefix(code, "```")
    code = strings.TrimSuffix(code, "```")
    return strings.TrimSpace(code)
}

func hasDirective(annotations []Annotation, directive Directive) bool {
    for _, ann := range annotations {
        if ann.Directive == directive {
            return true
        }
    }
    return false
}
```

---

## System Prompts

### Annotator System Prompt

```markdown
You are a Strudel code editor assistant. Your job is to analyze user requests and annotate existing code with precise editing directives.

Given:
1. User's request
2. Current editor state (Strudel code)
3. Conversation history (optional)

Output ONLY the annotated editor state with inline comments showing:
- `// MODIFY THIS: [reason]` - for code that should be changed
- `// DELETE THIS: [reason]` - for code that should be removed
- `// ADD NEW CODE BELOW: [reason]` - for where new code should go
- `// KEEP AS-IS` - for code that shouldn't change (optional, for clarity)
- `// REWRITE ENTIRE PATTERN: [reason]` - if complete restructure needed

Rules:
1. Be VERY specific about which lines to modify
2. Only annotate code that's relevant to the user's request
3. If the request is to add something new, mark where it should go
4. If the request is ambiguous, pick the most likely interpretation
5. Preserve all existing code structure - only add comment directives
6. Keep annotations brief and clear (10-30 words)
7. Never modify the actual code - only add directive comments

Examples:

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
User: "make the kick quieter"

Current code:
```
$: sound("bd*4")
$: sound("hh*8")
```

Output:
```
// MODIFY THIS: Reduce kick volume by adding .gain()
$: sound("bd*4")
$: sound("hh*8")
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
User: "remove the hi-hats"

Current code:
```
$: sound("bd*4")
$: sound("hh*8")
$: note("c2 e2").sound("sawtooth")
```

Output:
```
$: sound("bd*4")
// DELETE THIS: Remove hi-hat pattern as requested
$: sound("hh*8")
$: note("c2 e2").sound("sawtooth")
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
User: "add a melody"

Current code:
```
$: sound("bd*4")
$: sound("hh*8")
```

Output:
```
$: sound("bd*4")
$: sound("hh*8")
// ADD NEW CODE BELOW: Add a melodic pattern that complements the drums
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
User: "change bass to square wave"

Current code:
```
$: sound("bd*4")
$: note("c2").sound("sawtooth").cutoff(500)
```

Output:
```
$: sound("bd*4")
// MODIFY THIS: Change sawtooth to square wave
$: note("c2").sound("sawtooth").cutoff(500)
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Now annotate this code:
```

### Enhanced Generator Prompt (Phase 2)

When annotations are present, the generator's instructions are enhanced:

```markdown
!!! ANNOTATION DIRECTIVES - CRITICAL !!!

The CURRENT EDITOR STATE below has been ANNOTATED with editing directives.
You MUST follow these directives PRECISELY:

- `// MODIFY THIS: [reason]` → Change ONLY this specific line/block as described
- `// DELETE THIS: [reason]` → Remove this line/block entirely from your output
- `// ADD NEW CODE BELOW: [reason]` → Insert new code after this marker
- `// KEEP AS-IS` → Do NOT change this line at all
- `// REWRITE ENTIRE PATTERN: [reason]` → Complete restructure needed

CRITICAL RULES:
1. Only change code that has a directive comment above it
2. Preserve everything else EXACTLY as-is
3. Do NOT include the directive comments in your final output
4. Generate the complete, working Strudel code with changes applied

[Rest of normal instructions...]
```

---

## Decision Logic: When to Annotate

### Request Classification

Not all requests benefit from annotation. Implement a classifier to decide:

```go
// internal/agent/classifier.go
package agent

type RequestType int

const (
    RequestTypeNew      RequestType = iota  // Fresh generation
    RequestTypeAdd                          // Add to existing
    RequestTypeModify                       // Change existing - USE ANNOTATION
    RequestTypeDelete                       // Remove parts - USE ANNOTATION
    RequestTypeQuestion                     // Asking for help
)

// classifyRequest determines what type of request this is
func classifyRequest(query string, editorState string) RequestType {
    queryLower := strings.ToLower(query)

    // Check for questions first
    questionKeywords := []string{"how", "what", "why", "when", "can you explain"}
    for _, kw := range questionKeywords {
        if strings.HasPrefix(queryLower, kw) {
            return RequestTypeQuestion
        }
    }

    // Empty editor = new generation
    if strings.TrimSpace(editorState) == "" {
        return RequestTypeNew
    }

    // Check for modification keywords
    modifyKeywords := []string{
        "make", "change", "adjust", "tweak", "fix", "update",
        "modify", "edit", "alter", "convert", "replace",
    }
    for _, kw := range modifyKeywords {
        if strings.Contains(queryLower, kw) {
            return RequestTypeModify
        }
    }

    // Check for deletion keywords
    deleteKeywords := []string{
        "remove", "delete", "drop", "get rid of", "take out",
        "eliminate", "clear",
    }
    for _, kw := range deleteKeywords {
        if strings.Contains(queryLower, kw) {
            return RequestTypeDelete
        }
    }

    // Check for additive keywords
    addKeywords := []string{
        "add", "create", "make a", "build", "generate",
        "include", "insert", "also",
    }
    for _, kw := range addKeywords {
        if strings.Contains(queryLower, kw) {
            return RequestTypeAdd
        }
    }

    // Default to new if unclear
    return RequestTypeNew
}

// shouldUseAnnotation determines if annotation is beneficial
func shouldUseAnnotation(requestType RequestType, enabled bool) bool {
    if !enabled {
        return false
    }

    switch requestType {
    case RequestTypeModify, RequestTypeDelete:
        return true  // Annotation helps with precision
    case RequestTypeAdd, RequestTypeNew, RequestTypeQuestion:
        return false  // No benefit from annotation
    default:
        return false
    }
}
```

### Integration in Agent

```go
// internal/agent/agent.go

func (a *Agent) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
    // 1. Analyze query for actionability
    analysis, err := a.llm.AnalyzeQuery(ctx, req.UserQuery)
    if err != nil {
        return nil, fmt.Errorf("failed to analyze query: %w", err)
    }

    if !analysis.IsActionable {
        return &GenerateResponse{
            IsActionable:        false,
            ClarifyingQuestions: analysis.ClarifyingQuestions,
        }, nil
    }

    // 2. Classify request type
    requestType := classifyRequest(req.UserQuery, req.EditorState)

    // 3. Decide if annotation should be used
    useAnnotation := shouldUseAnnotation(requestType, a.config.EnableAnnotation)

    var editorState string
    var annotations []Annotation

    // 4. Annotate code if beneficial
    if useAnnotation && req.EditorState != "" {
        annotationReq := AnnotationRequest{
            UserQuery:           req.UserQuery,
            CurrentCode:         req.EditorState,
            ConversationHistory: req.Messages,
        }

        annotationResult, err := a.llm.AnnotateCode(ctx, annotationReq)
        if err != nil {
            logger.Warn("annotation failed, falling back to non-annotated flow", "error", err)
            editorState = req.EditorState
        } else if annotationResult.UseFallback {
            logger.Info("annotation confidence too low, using fallback", "confidence", annotationResult.Confidence)
            editorState = req.EditorState
        } else {
            logger.Info("using annotated code", "confidence", annotationResult.Confidence, "annotations", len(annotationResult.Annotations))
            editorState = annotationResult.AnnotatedCode
            annotations = annotationResult.Annotations
        }
    } else {
        editorState = req.EditorState
    }

    // 5. Continue with normal flow (search, prompt building, generation)
    // ... rest of existing implementation ...
}
```

---

## Confidence Scoring & Fallback

### Confidence Calculation

```go
func calculateConfidence(annotations []Annotation, totalLines int) float64 {
    if len(annotations) == 0 {
        return 0.0  // No annotations = no confidence
    }

    baseConfidence := 0.9

    // Factor 1: Too many annotations suggests uncertainty
    if len(annotations) > 3 {
        penalty := 0.1 * float64(len(annotations)-3)
        baseConfidence -= penalty
    }

    // Factor 2: Check reason quality
    vagueReasons := 0
    for _, ann := range annotations {
        if len(ann.Reason) < 10 {
            vagueReasons++
        }
    }
    if vagueReasons > 0 {
        baseConfidence -= 0.1 * float64(vagueReasons)
    }

    // Factor 3: Conflicting directives (e.g., MODIFY and DELETE on same target)
    if hasConflictingDirectives(annotations) {
        baseConfidence -= 0.3
    }

    // Clamp to [0.0, 1.0]
    if baseConfidence < 0.0 {
        baseConfidence = 0.0
    }
    if baseConfidence > 1.0 {
        baseConfidence = 1.0
    }

    return baseConfidence
}

func hasConflictingDirectives(annotations []Annotation) bool {
    // Check if same target appears in multiple directives
    targets := make(map[string]int)
    for _, ann := range annotations {
        if ann.Target != "" {
            targets[ann.Target]++
        }
    }

    for _, count := range targets {
        if count > 1 {
            return true
        }
    }

    return false
}
```

### Fallback Behavior

```go
// Confidence threshold configured in environment
const defaultConfidenceThreshold = 0.7

// In agent.Generate():
if annotationResult.Confidence < a.config.AnnotationConfidenceThreshold {
    logger.Warn("annotation confidence below threshold, falling back",
        "confidence", annotationResult.Confidence,
        "threshold", a.config.AnnotationConfidenceThreshold)

    // Use original, non-annotated code
    editorState = req.EditorState
    useAnnotation = false
}
```

---

## Configuration

### Environment Variables

```env
# Code Annotation Configuration
ENABLE_CODE_ANNOTATION=false
CODE_ANNOTATOR_PROVIDER=anthropic
CODE_ANNOTATOR_MODEL=claude-3-haiku-20240307
CODE_ANNOTATOR_MAX_TOKENS=500
CODE_ANNOTATOR_TEMPERATURE=0.2
ANNOTATION_CONFIDENCE_THRESHOLD=0.7
```

### Config Loading

```go
// internal/config/config.go

type Config struct {
    // ... existing fields ...

    // Annotation settings
    EnableAnnotation              bool
    AnnotatorProvider             llm.Provider
    AnnotatorModel                string
    AnnotatorMaxTokens            int
    AnnotatorTemperature          float32
    AnnotationConfidenceThreshold float64
}

func Load() (*Config, error) {
    return &Config{
        // ... existing fields ...

        EnableAnnotation:              getEnvBool("ENABLE_CODE_ANNOTATION", false),
        AnnotatorProvider:             llm.Provider(getEnv("CODE_ANNOTATOR_PROVIDER", "anthropic")),
        AnnotatorModel:                getEnv("CODE_ANNOTATOR_MODEL", "claude-3-haiku-20240307"),
        AnnotatorMaxTokens:            getEnvInt("CODE_ANNOTATOR_MAX_TOKENS", 500),
        AnnotatorTemperature:          getEnvFloat32("CODE_ANNOTATOR_TEMPERATURE", 0.2),
        AnnotationConfidenceThreshold: getEnvFloat64("ANNOTATION_CONFIDENCE_THRESHOLD", 0.7),
    }, nil
}
```

---

## Implementation Phases

### Phase 1: Enhanced Instructions (Quick Win) ✅ COMPLETED

**Goal:** Improve precision without adding complexity

**Tasks:**
1. ✅ Update `getInstructions()` in `internal/agent/prompt.go`
2. ✅ Add explicit guidance for ADD vs MODIFY vs DELETE operations
3. ✅ Include few-shot examples showing precise edits
4. ✅ Test with real user scenarios

**Timeline:** 1 day (actual)
**Cost:** $0 (no new infrastructure)
**Risk:** Low
**Status:** ✅ Completed 2025-12-28

**Success Criteria Met:**
- ✅ LLM correctly distinguishes ADD vs MODIFY requests 100% of the time (manual testing)
- ✅ Zero accidental modifications of unrelated code in all test cases
- ✅ User testing confirmed "surgical accuracy"

**Outcome:** Phase 1 is SUFFICIENT - no need to proceed to Phase 2

### Phase 2: Basic Annotation (On Hold - Not Currently Needed)

**Goal:** Implement core annotation system

**Status:** ⏸️ ON HOLD - Phase 1 achieved sufficient precision

**Tasks (if needed in future):**
1. Add `Annotator` interface to `internal/llm/types.go`
2. Implement `AnnotateCode()` in `internal/llm/anthropic.go`
3. Add request classification logic (`internal/agent/classifier.go`)
4. Integrate annotation into `agent.Generate()` flow
5. Add configuration options
6. Implement confidence scoring and fallback
7. Create comprehensive tests

**Estimated Timeline:** 1 week (if implemented)
**Estimated Cost:** +10% per request (~$0.0003 per annotation)
**Risk:** Medium (new moving part in pipeline)

**When to Reconsider:**
- Precision issues emerge over time
- Edge cases that Phase 1 can't handle
- User demand for change previews

**Success Criteria (if implemented):**
- Annotations correctly identify targets >85% of the time
- Confidence scoring prevents bad annotations
- Fallback works gracefully
- No degradation in code quality when annotation is used

### Phase 3: Advanced Features (Future)

**Goal:** Polish and enhance annotation capabilities

**Tasks:**
1. Multi-option annotations ("Option 1:", "Option 2:")
2. Annotation preview UI (show user what will change)
3. User feedback loop (accept/reject annotations)
4. Metrics tracking (accuracy, user acceptance rate)
5. A/B testing (annotated vs non-annotated)

**Timeline:** 2-3 weeks
**Cost:** TBD based on Phase 2 learnings
**Risk:** Low (incremental improvements)

---

## Testing Strategy

### Unit Tests

```go
// internal/llm/anthropic_test.go

func TestAnnotateCode_ModifyRequest(t *testing.T) {
    tests := []struct {
        name          string
        userQuery     string
        currentCode   string
        wantDirective Directive
        wantTarget    string
    }{
        {
            name:          "make kick quieter",
            userQuery:     "make the kick quieter",
            currentCode:   "$: sound(\"bd*4\")\n$: sound(\"hh*8\")",
            wantDirective: DirectiveModify,
            wantTarget:    "sound(\"bd*4\")",
        },
        {
            name:          "remove hi-hats",
            userQuery:     "remove the hi-hats",
            currentCode:   "$: sound(\"bd*4\")\n$: sound(\"hh*8\")",
            wantDirective: DirectiveDelete,
            wantTarget:    "sound(\"hh*8\")",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Mock LLM or use real API in integration tests
            result, err := annotator.AnnotateCode(ctx, AnnotationRequest{
                UserQuery:   tt.userQuery,
                CurrentCode: tt.currentCode,
            })

            assert.NoError(t, err)
            assert.NotNil(t, result)
            assert.Greater(t, result.Confidence, 0.7)
            assert.Contains(t, result.Annotations, Annotation{
                Directive: tt.wantDirective,
                Target:    tt.wantTarget,
            })
        })
    }
}
```

### Integration Tests

```go
// internal/agent/agent_integration_test.go

func TestGenerateWithAnnotation(t *testing.T) {
    agent := NewAgent(config)

    // test MODIFY request
    resp, err := agent.Generate(ctx, GenerateRequest{
        UserQuery:   "make the kick quieter",
        EditorState: "$: sound(\"bd*4\")\n$: sound(\"hh*8\")",
    })

    assert.NoError(t, err)
    assert.Contains(t, resp.Code, "gain(")  // Should add gain
    assert.Contains(t, resp.Code, "hh*8")   // Should preserve hi-hats
}
```

### Manual Testing Scenarios

1. **Modification Precision**
   - Request: "make bass darker"
   - Verify: Only bass line modified, drums unchanged

2. **Deletion Accuracy**
   - Request: "remove the melody"
   - Verify: Melody removed, drums and bass preserved

3. **Addition Placement**
   - Request: "add a snare"
   - Verify: Snare added in logical place, existing patterns unchanged

4. **Fallback Behavior**
   - Provide ambiguous request
   - Verify: System falls back to non-annotated flow gracefully

---

## Metrics & Monitoring

### Key Metrics

```go
type AnnotationMetrics struct {
    // Performance
    AnnotationLatency   time.Duration  // How long annotation takes
    ConfidenceAverage   float64        // Average confidence score
    FallbackRate        float64        // % of times we fall back

    // Accuracy
    CorrectAnnotations  int            // Manually verified correct
    IncorrectAnnotations int           // Manually verified incorrect
    AccuracyRate        float64        // Correct / (Correct + Incorrect)

    // Usage
    AnnotationsUsed     int            // Times annotation was used
    AnnotationsSkipped  int            // Times we skipped annotation

    // Cost
    AnnotationCost      float64        // Total cost of annotations
}
```

### Logging

```go
logger.Info("annotation completed",
    "query", req.UserQuery,
    "confidence", result.Confidence,
    "annotations_count", len(result.Annotations),
    "fallback", result.UseFallback,
    "latency_ms", latency.Milliseconds(),
)
```

---

## Benefits Recap

| Benefit | Description | Impact |
|---------|-------------|--------|
| **Precision** | Changes only what user requests | High |
| **Preservation** | Prevents accidental modifications | High |
| **Cost** | Only +10% increase | Low |
| **Debuggable** | Can show annotations to user | Medium |
| **Confidence-aware** | Falls back when uncertain | High |
| **Iterative UX** | Natural refinement workflow | High |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Annotator makes mistakes | Medium | High | Confidence scoring + fallback |
| Added latency | Low | Medium | Use Haiku (fast), run in parallel with search |
| Configuration complexity | Low | Low | Sensible defaults, clear docs |
| User confusion | Low | Low | Don't expose annotations in UI initially |
| Cost increase | High | Low | Only 10%, can be toggled off |

---

## Future Enhancements

1. **Visual Annotation Preview**
   - Show user diff of what will change before generating
   - Allow user to approve/reject annotations

2. **Learning from Feedback**
   - Track which annotations users accept/reject
   - Fine-tune annotator prompts based on patterns

3. **Multi-Model Ensemble**
   - Use multiple LLMs for annotation
   - Combine results for higher confidence

4. **Contextual Confidence Tuning**
   - Adjust threshold based on request complexity
   - Lower threshold for simple requests, higher for complex

---

## References

- [RAG_ARCHITECTURE.md](./RAG_ARCHITECTURE.md) - Overall system architecture
- [HYBRID_RETRIEVAL_GUIDE.md](./HYBRID_RETRIEVAL_GUIDE.md) - Retrieval implementation
- `/internal/llm/types.go` - LLM interface definitions
- `/internal/agent/agent.go` - Agent orchestration
- `/internal/agent/prompt.go` - Prompt construction

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-12-28 | Initial specification |
| 1.1 | 2025-12-28 | Added Phase 1 implementation status and test results. Phase 2 marked as on hold - not needed. |

---

**End of Specification**
