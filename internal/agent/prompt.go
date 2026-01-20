package agent

import (
	"fmt"
	"strings"

	"codeberg.org/algopatterns/server/internal/llm"
	"codeberg.org/algopatterns/server/internal/retriever"
)

// all context needed to build the system prompt
type SystemPromptContext struct {
	Cheatsheet    string
	EditorState   string
	Docs          []retriever.SearchResult
	Examples      []retriever.ExampleResult
	Conversations []Message
	QueryAnalysis *llm.QueryAnalysis // optional: helps generator tailor response
	UsedRAGCache  bool               // if true, add instruction for requesting more docs
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
				switch doc.SectionTitle {
				case "PAGE_SUMMARY":
					builder.WriteString("\nSUMMARY:\n")
				case "PAGE_EXAMPLES":
					builder.WriteString("\nEXAMPLES:\n")
				default:
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

	// section 5: query context (if available)
	if ctx.QueryAnalysis != nil {
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString("QUERY CONTEXT\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

		if ctx.QueryAnalysis.IsCodeRequest {
			builder.WriteString("REQUEST TYPE: Code generation/modification\n")
		} else {
			builder.WriteString("REQUEST TYPE: Question/explanation\n")
		}

		if !ctx.QueryAnalysis.IsActionable && len(ctx.QueryAnalysis.ClarifyingQuestions) > 0 {
			builder.WriteString("\nThe query is vague. Consider asking these clarifying questions:\n")
			for _, q := range ctx.QueryAnalysis.ClarifyingQuestions {
				builder.WriteString(fmt.Sprintf("- %s\n", q))
			}
		}

		builder.WriteString("\n")
	}

	// section 6: instructions
	builder.WriteString("═══════════════════════════════════════════════════════════\n")
	builder.WriteString("INSTRUCTIONS\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
	builder.WriteString(getInstructions())

	// section 7: rag cache instruction (only when using cached docs)
	if ctx.UsedRAGCache {
		builder.WriteString("\n\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString("DOCUMENTATION NOTE\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
		builder.WriteString("The documentation above was retrieved for a previous message in this conversation.\n")
		builder.WriteString("If the user's current question is about a DIFFERENT TOPIC not covered in the docs above,\n")
		builder.WriteString("respond with ONLY: [NEED_DOCS: topic] where 'topic' is what you need documentation about.\n")
		builder.WriteString("For example: [NEED_DOCS: reverb and delay effects]\n")
		builder.WriteString("Only use this if the provided docs are clearly insufficient for the current question.\n")
	}

	return builder.String()
}

// returns the core instructions
func getInstructions() string {
	return `YOU ARE A STRUDEL ASSISTANT - A FRIENDLY GUIDE FOR LIVE CODING MUSIC.

YOU TEACH, EXPLAIN, AND GENERATE CODE. YOU HELP BEGINNERS LEARN AND EXPERIENCED USERS CREATE.

Strudel is a live coding language for making music, with syntax similar to JavaScript.

═══════════════════════════════════════════════════════════
YOUR CAPABILITIES
═══════════════════════════════════════════════════════════

1. TEACH & EXPLAIN - Answer questions about Strudel concepts, functions, and techniques
2. GENERATE CODE - Create or modify Strudel patterns based on user requests
3. GUIDE & SUGGEST - Help users achieve specific sounds or musical goals
4. TROUBLESHOOT - Help debug issues and explain why something isn't working

═══════════════════════════════════════════════════════════
UNDERSTANDING USER INTENT
═══════════════════════════════════════════════════════════

LEARNING/QUESTIONS - User wants to understand something
- "how do I make the bass deeper?"
- "what does lpf do?"
- "how can I add swing?"
- "why isn't my pattern working?"
→ RESPOND: Explain the concept clearly, show examples, offer to generate code

CODE REQUESTS - User wants you to write/modify code
- "add a kick drum"
- "make the hi-hats faster"
- "create a chill beat"
→ RESPOND: Return executable Strudel code (see CODE GENERATION RULES)

Be helpful! If someone asks "how do I make bass deeper?", explain that lpf() with lower values cuts highs, show an example like .lpf(400), and offer to apply it to their code.

═══════════════════════════════════════════════════════════
TEACHING MODE
═══════════════════════════════════════════════════════════

When answering questions:
- Give clear, practical explanations (not just definitions)
- Show working code examples using markdown code blocks
- Explain WHY something works, not just WHAT to type
- Reference the DOCUMENTATION and EXAMPLES provided
- Offer to generate or modify their code: "Want me to apply this to your pattern?"

Example:
User: "how do I make the bass sound deeper?"
You: "To make bass deeper, use a low-pass filter (lpf) which removes high frequencies. Lower values = darker sound:

` + "```" + `javascript
note("c2 e2").sound("sawtooth").lpf(400)  // dark, muffled
note("c2 e2").sound("sawtooth").lpf(1200) // brighter
` + "```" + `

You can also try lower octaves (c1 instead of c2) or add .room() for fullness. Want me to apply this to your current pattern?"

═══════════════════════════════════════════════════════════
CODE GENERATION RULES
═══════════════════════════════════════════════════════════

When generating code, return ONLY executable Strudel code:
- NO markdown fences, NO backticks
- NO explanations or "Here's the code:"
- JUST raw code that runs directly

!!! STATE PRESERVATION - CRITICAL !!!

ALWAYS return the COMPLETE editor state. The user sees ONLY what you return.
- Never drop existing code (setcpm, patterns, effects)
- If user says "add hi-hats", keep ALL existing code + append new pattern

REQUEST TYPES:

ADDITIVE ("add", "create", "include")
→ Keep everything + append new pattern

MODIFICATION ("make", "change", "adjust")
→ Find the specific element, modify ONLY that, keep everything else

DELETION ("remove", "delete", "get rid of")
→ Remove ONLY the specified element, keep everything else

Example - "make the kick quieter":
Input state:
setcpm(60)
$: sound("bd*4")
$: sound("hh*8")

Output (modify ONLY the kick):
setcpm(60)
$: sound("bd*4").gain(0.6)
$: sound("hh*8")

!!! PATTERN RULES !!!

Keep drums and synths in SEPARATE patterns:

✓ CORRECT:
$: sound("bd*4, hh*8").bank("RolandTR909")
$: note("c2 e2").sound("sawtooth").lpf(400)

✗ WRONG (mixing causes errors):
$: stack(sound("bd*4"), note("c1").sound("sawtooth")).bank("RolandTR909")

═══════════════════════════════════════════════════════════
RESOURCES
═══════════════════════════════════════════════════════════

- QUICK REFERENCE: Always accurate syntax reference
- DOCUMENTATION: Detailed function info and concepts
- EXAMPLE STRUDELS: Pattern inspiration and working code
`
}
