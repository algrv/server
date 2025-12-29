package strudel

import (
	"regexp"
	"slices"
	"strings"
)

// sound sample definitions organized by category (source of truth)
var soundDefs = SoundDefinitions{
	Drums: []string{
		"bd", "sd", "rim", "cp", "hh", "oh", "cr", "rd", "ht", "mt", "lt",
	},

	Percussion: []string{
		"bd", "sd", "rim", "cp", "hh", "oh", "cr", "rd", "ht", "mt", "lt",
		"sh", "cb", "tb", "perc",
	},

	Synth: []string{
		"sine", "sawtooth", "square", "triangle",
	},

	Noise: []string{
		"white", "pink", "brown", "crackle",
	},

	ZZFX: []string{
		"z_sawtooth", "z_tan", "z_noise", "z_sine", "z_square",
	},

	Wavetable: []string{
		"wt_", // prefix match
	},

	Misc: []string{
		"misc", "fx",
	},

	Custom: []string{
		"user",
	},
}

// builds lookup map from sound definitions
func buildSoundLookup() map[string][]string {
	return map[string][]string{
		"drums":      soundDefs.Drums,
		"percussion": soundDefs.Percussion,
		"synth":      soundDefs.Synth,
		"noise":      soundDefs.Noise,
		"zzfx":       soundDefs.ZZFX,
		"wavetable":  soundDefs.Wavetable,
		"misc":       soundDefs.Misc,
		"custom":     soundDefs.Custom,
	}
}

// sound categorization lookup (built from soundDefs)
var soundCategories = buildSoundLookup()

// effect function definitions organized by category
var effectDefs = EffectDefinitions{
	Filter: []string{
		"lpf", "hpf", "bpf", "lpq", "hpq", "bpq", "vowel", "ftype",
	},

	FilterEnvelope: []string{
		"lpattack", "lpdecay", "lpsustain", "lprelease", "lpenv",
		"lpa", "lpd", "lps", "lpr", "lpe",
		"hpattack", "hpdecay", "hpsustain", "hprelease", "hpenv",
		"bpattack", "bpdecay", "bpsustain", "bprelease", "bpenv",
	},

	Distortion: []string{
		"coarse", "crush", "distort", "shape",
	},

	Dynamics: []string{
		"gain", "velocity", "compressor", "postgain", "post", "xfade",
	},

	Spatial: []string{
		"pan", "jux", "juxBy",
	},

	Delay: []string{
		"delay", "delaytime", "delayfeedback", "echo",
	},

	Reverb: []string{
		"room", "roomsize", "roomfade", "roomlp", "roomdim", "iresponse",
	},

	Modulation: []string{
		"phaser", "phaserdepth", "phasercenter", "phasersweep",
		"tremolo", "tremolosync", "tremolodepth", "tremoloskew",
		"tremolophase", "tremoloshape",
		"vib", "vibmod", "am",
	},

	Envelope: []string{
		"attack", "decay", "dec", "sustain", "release", "adsr",
	},

	PitchEnvelope: []string{
		"pattack", "pdecay", "prelease", "penv", "pcurve", "panchor",
	},

	FMSynthesis: []string{
		"fm", "fmh", "fmattack", "fmdecay", "fmsustain", "fmenv",
	},

	Sampler: []string{
		"begin", "end", "loop", "loopBegin", "loopEnd",
		"cut", "clip", "loopAt", "fit",
		"chop", "striate", "slice", "splice", "scrub", "speed",
	},

	Routing: []string{
		"orbit", "duckorbit",
	},

	Sidechain: []string{
		"duck", "duckattack", "duckdepth",
	},

	Synthesis: []string{
		"partials", "phases", "noise",
	},

	ZZFX: []string{
		"zrand", "curve", "slide", "deltaSlide", "zmod", "zcrush",
		"zdelay", "pitchJump", "pitchJumpTime", "lfo",
	},
}

// builds lookup map from effect definitions
func buildEffectLookup() map[string]string {
	lookup := make(map[string]string)

	addToLookup := func(functions []string, category string) {
		for _, fn := range functions {
			lookup[fn] = category
		}
	}

	addToLookup(effectDefs.Filter, "filter")
	addToLookup(effectDefs.FilterEnvelope, "filter-envelope")
	addToLookup(effectDefs.Distortion, "distortion")
	addToLookup(effectDefs.Dynamics, "dynamics")
	addToLookup(effectDefs.Spatial, "spatial")
	addToLookup(effectDefs.Delay, "delay")
	addToLookup(effectDefs.Reverb, "reverb")
	addToLookup(effectDefs.Modulation, "modulation")
	addToLookup(effectDefs.Envelope, "envelope")
	addToLookup(effectDefs.PitchEnvelope, "pitch-envelope")
	addToLookup(effectDefs.FMSynthesis, "fm-synthesis")
	addToLookup(effectDefs.Sampler, "sampler")
	addToLookup(effectDefs.Routing, "routing")
	addToLookup(effectDefs.Sidechain, "sidechain")
	addToLookup(effectDefs.Synthesis, "synthesis")
	addToLookup(effectDefs.ZZFX, "zzfx")

	return lookup
}

// effect categorization lookup
var effectCategories = buildEffectLookup()

// performs full semantic analysis on Strudel code
func AnalyzeCode(code string) CodeAnalysis {
	parsed := Parse(code)

	analysis := CodeAnalysis{
		SoundTags:      analyzeSounds(code, parsed),
		EffectTags:     analyzeEffects(parsed),
		MusicalTags:    analyzeMusicalElements(code, parsed),
		ComplexityTags: analyzeComplexityTags(code, parsed),
		Complexity:     calculateComplexity(code, parsed),
		LineCount:      strings.Count(code, "\n") + 1,
		FunctionCount:  len(parsed.Functions),
		VariableCount:  len(parsed.Variables),
	}

	return analysis
}

// analyzeSounds categorizes sounds into semantic tags
func analyzeSounds(code string, parsed ParsedCode) []string {
	tags := make(map[string]bool)

	// check each sound against categories
	for _, sound := range parsed.Sounds {
		for category, sounds := range soundCategories {
			if contains(sounds, sound) {
				tags[category] = true
			}
		}

		// check for wavetable prefix
		if strings.HasPrefix(sound, "wt_") {
			tags["wavetable"] = true
		}
	}

	// check for bass pattern in code (case-insensitive)
	if regexp.MustCompile(`(?i)bass`).MatchString(code) {
		tags["bass"] = true
	}

	// convert map to slice
	return mapKeysToSlice(tags)
}

// identifies audio effects used in the code
func analyzeEffects(parsed ParsedCode) []string {
	tags := make(map[string]bool)

	// check each function against effect category mappings
	for _, function := range parsed.Functions {
		if effectTag, exists := effectCategories[function]; exists {
			tags[effectTag] = true
		}
	}

	return mapKeysToSlice(tags)
}

// analyzeMusicalElements identifies musical constructs
func analyzeMusicalElements(code string, parsed ParsedCode) []string {
	tags := make(map[string]bool)

	// note patterns indicate melody
	if len(parsed.Notes) > 0 {
		tags["melody"] = true
		tags["melodic"] = true
	}

	// scale usage
	if len(parsed.Scales) > 0 || contains(parsed.Functions, "scale") {
		tags["scales"] = true
		tags["melodic"] = true
	}

	// chord patterns (multiple notes at once indicated by comma)
	if regexp.MustCompile(`note\s*\(\s*["'][^"']*,`).MatchString(code) {
		tags["chords"] = true
		tags["harmony"] = true
	}

	// rhythm patterns
	if contains(parsed.Functions, "fast") || contains(parsed.Functions, "slow") {
		tags["rhythm"] = true
	}

	// sequences (note patterns)
	if regexp.MustCompile(`note\s*\(\s*["'][a-g0-9\s]+["']`).MatchString(code) {
		tags["sequences"] = true
	}

	return mapKeysToSlice(tags)
}

// analyzeComplexityTags generates complexity-related tags
func analyzeComplexityTags(code string, parsed ParsedCode) []string {
	tags := []string{}

	stackCount := parsed.Patterns["stack"]
	varCount := len(parsed.Variables)

	// layering tags
	if stackCount > 3 {
		tags = append(tags, "complex", "layered")
	} else if stackCount > 0 {
		tags = append(tags, "layered")
	}

	// structure tags
	if parsed.Patterns["arrange"] > 0 {
		tags = append(tags, "arranged", "structured")
	}

	// advanced code tags
	if varCount > 5 {
		tags = append(tags, "advanced")
	}

	// interactive tags
	if parsed.Patterns["slider"] > 0 {
		tags = append(tags, "interactive")
	}

	// simple/beginner-friendly tags
	if len(code) < 200 && stackCount == 0 && varCount == 0 {
		tags = append(tags, "simple", "beginner-friendly")
	}

	return tags
}

// returns a 0-10 complexity score
func calculateComplexity(code string, parsed ParsedCode) int {
	score := 0

	// base complexity from code length
	if len(code) > 500 {
		score += 3
	} else if len(code) > 200 {
		score += 2
	} else {
		score++
	}

	// layering complexity
	stackCount := parsed.Patterns["stack"]
	if stackCount > 3 {
		score += 3
	} else if stackCount > 0 {
		score += 2
	}

	// variable usage
	varCount := len(parsed.Variables)
	if varCount > 5 {
		score += 2
	} else if varCount > 0 {
		score++
	}

	// advanced patterns
	if parsed.Patterns["arrange"] > 0 {
		score += 2
	}

	// interactive elements
	if parsed.Patterns["slider"] > 0 {
		score++
	}

	// cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

// combines analysis with existing metadata to create final tag list
func GenerateTags(analysis CodeAnalysis, category string, existingTags []string) []string {
	tags := make(map[string]bool)

	// start with existing tags
	for _, tag := range existingTags {
		if tag != "" {
			tags[strings.ToLower(tag)] = true
		}
	}

	// add category as tag
	if category != "" {
		tags[strings.ToLower(category)] = true
	}

	// add all analysis tags
	for _, tag := range analysis.SoundTags {
		tags[tag] = true
	}

	for _, tag := range analysis.EffectTags {
		tags[tag] = true
	}

	for _, tag := range analysis.MusicalTags {
		tags[tag] = true
	}

	for _, tag := range analysis.ComplexityTags {
		tags[tag] = true
	}

	return mapKeysToSlice(tags)
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func mapKeysToSlice(m map[string]bool) []string {
	result := make([]string, 0, len(m))

	for key := range m {
		result = append(result, key)
	}

	return result
}
