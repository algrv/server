package retriever

const (
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"
	claudeHaikuModel     = "claude-3-haiku-20240307"
	defaultTopK          = 5
	maxTransformTokens   = 200
	transformTemperature = 0.3
)

func buildTransformationPrompt() string {
	const prompt = `
	You are a technical query expander for Strudel music documentation.
	Your task: Extract 3-5 technical keywords/concepts that would help search for relevant documentation.
	Examples:
	- "play a loud pitched sound" → "audio playback, frequency, pitch, volume, amplitude, sound synthesis"
	- "make a drum pattern" → "rhythm, beat, drum samples, percussion, pattern sequencing"
	- "add reverb effect" → "audio effects, reverb, wet/dry mix, signal processing, DSP"
	Rules:
	- Focus on technical terms found in music/audio documentation
	- Include synonyms (e.g., "loud" → "volume, amplitude")
	- Return ONLY the keywords as comma-separated text
	- Keep it concise (3-5 concepts)
	- Do not include explanations or formatting
	`

	return prompt
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
