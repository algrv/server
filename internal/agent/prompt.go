package agent

import (
	"fmt"
	"strings"

	"github.com/algorave/server/internal/retriever"
)

// holds all the context needed to build the system prompt
type SystemPromptContext struct {
	Cheatsheet    string
	EditorState   string
	Docs          []retriever.SearchResult
	Examples      []retriever.ExampleResult
	Conversations []Message
}

// assembles the complete system prompt
func buildSystemPrompt(ctx SystemPromptContext) string {
	var builder strings.Builder

	// section 1: cheatsheet (always accurate - use this first)
	builder.WriteString("═══════════════════════════════════════════════════════════\n")
	builder.WriteString("STRUDEL QUICK REFERENCE (ALWAYS ACCURATE - USE THIS FIRST)\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
	builder.WriteString(ctx.Cheatsheet)
	builder.WriteString("\n\n")

	// section 2: current editor state
	if ctx.EditorState != "" {
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString("CURRENT EDITOR STATE\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
		builder.WriteString(ctx.EditorState)
		builder.WriteString("\n\n")
	}

	// section 3: relevant documentation (if any)
	if len(ctx.Docs) > 0 {
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString("RELEVANT DOCUMENTATION (Technical + Concepts)\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

		// group docs by page
		pageMap := make(map[string][]retriever.SearchResult)
		pageOrder := []string{}

		for _, doc := range ctx.Docs {
			if _, exists := pageMap[doc.PageName]; !exists {
				pageOrder = append(pageOrder, doc.PageName)
			}
			pageMap[doc.PageName] = append(pageMap[doc.PageName], doc)
		}

		// render docs grouped by page
		for _, pageName := range pageOrder {
			builder.WriteString("─────────────────────────────────────────\n")
			builder.WriteString(fmt.Sprintf("Page: %s\n", pageName))
			builder.WriteString("─────────────────────────────────────────\n")

			for _, doc := range pageMap[pageName] {
				if doc.SectionTitle == "PAGE_SUMMARY" {
					builder.WriteString("\nSUMMARY:\n")
				} else if doc.SectionTitle == "PAGE_EXAMPLES" {
					builder.WriteString("\nEXAMPLES:\n")
				} else {
					builder.WriteString(fmt.Sprintf("\nSECTION: %s\n", doc.SectionTitle))
				}

				builder.WriteString(doc.Content)
				builder.WriteString("\n")
			}

			builder.WriteString("\n")
		}
	}

	// section 4: example strudels (if any)
	if len(ctx.Examples) > 0 {
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString("EXAMPLE STRUDELS FOR REFERENCE\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

		for i, example := range ctx.Examples {
			builder.WriteString("─────────────────────────────────────────\n")
			builder.WriteString(fmt.Sprintf("Example %d: %s\n", i+1, example.Title))

			if example.Description != "" {
				builder.WriteString(fmt.Sprintf("Description: %s\n", example.Description))
			}

			if len(example.Tags) > 0 {
				builder.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(example.Tags, ", ")))
			}

			builder.WriteString("─────────────────────────────────────────\n")
			builder.WriteString(example.Code)
			builder.WriteString("\n\n")
		}
	}

	// section 5: instructions
	builder.WriteString("═══════════════════════════════════════════════════════════\n")
	builder.WriteString("INSTRUCTIONS\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
	builder.WriteString(getInstructions())

	return builder.String()
}

// returns the core instructions
func getInstructions() string {
	return `You are a Strudel code generation assistant.

	Strudel is a special programming language for live coding music. It's a port of Tidal Cycles to JavaScript.

	Your task is to generate Strudel code based on the user's request.

	Guidelines:
	- Use the QUICK REFERENCE for accurate syntax (it's always correct)
	- Build upon the CURRENT EDITOR STATE when the user asks to modify existing code
	- Reference the DOCUMENTATION for detailed information about functions and concepts
	- Reference the EXAMPLE STRUDELS for pattern inspiration
	- Return ONLY executable Strudel code unless the user explicitly asks for an explanation
	- Keep code concise and focused on the user's request
	- Use comments sparingly and only when the code logic isn't self-evident

	!!! BE MINIMAL AND LITERAL !!!

	- When CURRENT EDITOR STATE exists: ALWAYS return the COMPLETE code with your changes
	- PRESERVE all existing patterns/code - don't drop anything unless explicitly asked to remove it
	- ONLY add/modify what the user explicitly requests - nothing more
	- Do NOT anticipate or add extra features the user didn't request
	- Examples:
	  * "set BPM to 120" (no editor state) → ONLY: setcpm(60)
	  * "add a kick" (editor has: setcpm(60)) → RETURN: setcpm(60) + $: sound("bd*4")
	  * "add hi-hats" (editor has BPM + kick) → RETURN: setcpm(60) + kick + new hi-hats
	  * "change hi-hats to 16 times" → RETURN: full code with ONLY hi-hats modified
	- Wait for the user to request each element instead of creating a complete track
	- Most requests require simple patterns like note("c e g") or sound("bd") at a time

	!!! CRITICAL PATTERN RULES !!!

	NEVER mix different sound types in the same stack() call.
	Keep drums, synths, and melodies in SEPARATE patterns.

	✓ CORRECT (separate patterns for different sound types):
	$: sound("bd*4, hh*8").bank("RolandTR909")
	$: note("c1 e1 g1").sound("sawtooth").lpf(400)

	✓ ALSO CORRECT (using variables, then stacking):
	let drums = sound("bd*4, hh*8").bank("RolandTR909")
	let bass = note("c1 e1 g1").sound("sawtooth").lpf(400)
	$: stack(drums, bass)

	✗ WRONG (mixing drums and synths in same stack - will cause errors):
	$: stack(
	  sound("bd*4"),
	  note("c1").sound("sawtooth")
	).bank("RolandTR909")

	Rule: One stack = one sound type. Drums with drums, synths with synths.

	Response format:
	- For code requests: Return ONLY the code, no markdown formatting, no explanations
	- For questions: Provide a brief answer, then offer to generate code if relevant
`
}
