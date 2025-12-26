# 🌀 STRUDEL QUICK REFERENCE CHEATSHEET

A complete reference guide to the Strudel live coding music language - an official port of Tidal Cycles to JavaScript.

---

## TABLE OF CONTENTS

1. [Quick Start](#quick-start)
2. [Strudel Editor Syntax](#strudel-editor-syntax-important)
3. [Mini-Notation Syntax](#mini-notation-syntax)
4. [Sound Functions](#sound-functions)
5. [Note & Pitch Functions](#note--pitch-functions)
6. [Audio Effects](#audio-effects)
7. [Time Modifiers](#time-modifiers)
8. [Random Modifiers](#random-modifiers)
9. [Tonal Functions](#tonal-functions)
10. [Synths & Oscillators](#synths--oscillators)
11. [Sampler Effects](#sampler-effects)
12. [Pattern Functions](#pattern-functions)
13. [Tips & Best Practices](#tips--best-practices)

---

## BASIC IDEAS (SOUNDS, NOTES & TEMPO)

### Play Your First Sound
```javascript
sound("casio")                    // Play a sound
sound("bd hh sd hh")              // Play sequence
sound("bd*4, hh*8")               // Parallel patterns (comma)
```

### Play Your First Notes
```javascript
note("c e g b")                   // Play notes by letter
note("48 52 55 59")               // Play notes by MIDI number
note("c4 d4 e4 f4")               // Play with octave
```

### Set Tempo
```javascript
setcpm(120)                       // Cycles per minute
```

### Stop Everything
```javascript
hush()                            // Stop all patterns
```

---

## STRUDEL EDITOR SYNTAX (IMPORTANT!)

**When writing code in the Strudel editor, every pattern MUST start with `$:` or `$<name>:`**

This is a requirement of the Strudel REPL editor. Without this prefix, your patterns will not play.

### Basic Pattern Prefix
```javascript
$: sound("bd sd")                 // Single pattern
$: note("c e g").sound("piano")   // Pattern with method chain
$: s("bd*4").gain(0.8)            // Using shorthand (s = sound)
```

### Named Patterns
```javascript
$drums: sound("bd sd hh cp")      // Named pattern
$bass: note("c2 e2").sound("sawtooth")
$melody: n("0 2 4 7").scale("C:minor").sound("piano")
```

### Multiple Patterns Running Simultaneously
```javascript
$: sound("bd*4")                  // Kick drum
$: sound("hh*8")                  // Hi-hats
$: note("c e g").sound("piano")   // Piano chords
```

### Stack Multiple Patterns in One Line
```javascript
$: stack(
  sound("bd*4"),
  sound("hh*8"),
  note("c e g").sound("piano")
)
```

### Muting Patterns
```javascript
_$: sound("bd")                   // Mute with _ prefix
_$drums: sound("bd sd")           // Mute named pattern
```

### Pattern Control Commands
```javascript
hush()                            // Stop all patterns
setcpm(120)                       // Set tempo (cycles per minute)
```

**Note**: The examples in this cheatsheet show the pattern code WITHOUT the `$:` prefix for clarity. Remember to add `$:` or `$<name>:` when using them in the Strudel editor!

---

## MINI-NOTATION SYNTAX

The Mini-Notation is a custom language for writing rhythmic patterns.

### Basic Sequences
```javascript
sound("bd sd hh cp")              // Space-separated sequence
sound("bd ~ sd ~")                // ~ = rest/silence
sound("bd - sd -")                // - = rest (alternative)
```

### Timing Control

| Syntax | Description | Example |
|--------|-------------|---------|
| `<a b c>` | Play one per cycle | `sound("<bd hh sd>")` |
| `[a b c]/2` | Play over 2 cycles | `sound("[bd hh sd]/2")` |
| `[a b c]*2` | Play twice per cycle | `sound("[bd hh sd]*2")` |
| `a@2` | Elongate (weight=2) | `sound("bd@2 hh")` |
| `a!2` | Replicate (no speedup) | `sound("bd!2 hh")` |

### Nested Sequences (Sub-Sequences)
```javascript
sound("bd [hh hh] sd [hh bd]")   // [brackets] subdivide time
sound("bd [[hh sd] cp] rim")      // [[double brackets]] for deeper nesting
```

### Parallel (Polyphony)
```javascript
sound("bd, hh*8, sd")             // Comma = play simultaneously
note("c e g, c2 e2 g2")           // Multiple parallel patterns
```

### Euclidean Rhythms
```javascript
sound("bd(3,8)")                  // 3 beats over 8 steps
sound("bd(3,8,2)")                // With offset parameter
```

### Randomness
```javascript
note("[c e g]?")                  // 50% chance of removal
note("[c e g]?0.2")               // 20% chance of removal
note("[c|e|g]")                   // Choose randomly between c, e, or g
```

### Sample Selection
```javascript
sound("hh:0 hh:1 hh:2 hh:3")     // Select sample number
n("0 1 [4 2] 3*2").sound("jazz")  // Using n() function
```

---

## SOUND FUNCTIONS

### Playing Sounds
```javascript
sound("bd")                       // Play by name (shorthand: s)
s("bd")                           // Shorthand
s("bd sd [~ bd] sd")              // Sequence
```

### Default Sound Banks

**Drum Sounds:**
```
bd = bass drum
sd = snare drum  
rim = rimshot
cp = clap
hh = closed hi-hat
oh = open hi-hat
ht = high tom
mt = middle tom
lt = low tom
rd = ride cymbal
cr = crash cymbal
```

**Other Percussive:**
```
sh = shaker
cb = cowbell
tb = tambourine
perc = misc percussion
misc = miscellaneous samples
```

### Changing Sound Banks
```javascript
sound("bd hh sd oh").bank("RolandTR909")
sound("bd hh sd").bank("<RolandTR808 RolandTR909>")  // Pattern of banks
```

### Available Drum Machines
```
RolandTR808        RolandTR909
RolandTR707        RolandTR505
AkaiLinn           RhythmAce
ViscoSpaceDrum
```

### Custom Sample Loading
```javascript
// Load from URLs
samples({
  bassdrum: 'bd/BT0AADA.wav',
  hihat: 'hh27/000_hh27closedhh.wav',
  snaredrum: ['sd/rytm-01-classic.wav', 'sd/rytm-00-hard.wav']
}, 'https://raw.githubusercontent.com/tidalcycles/Dirt-Samples/master/')

// Load from GitHub
samples('github:tidalcycles/dirt-samples')

// From strudel.json
samples('https://raw.githubusercontent.com/tidalcycles/Dirt-Samples/master/strudel.json')
```

### Sample Aliases
```javascript
soundAlias('RolandTR808_bd', 'kick')
s("kick")
```

---

## NOTE & PITCH FUNCTIONS

### Playing Notes
```javascript
note("c e g b")                   // By letter
note("48 52 55 59")               // By MIDI number
note("c2 e3 g4 b5")               // With octave (default: octave 3)
```

### Note Notation
```javascript
note("db eb gb ab bb")            // Flats
note("c# d# f# g# a#")            // Sharps
```

### Scales
```javascript
n("0 2 4 6").scale("C:major")     // Numbers become scale degrees
n("<0 1 2 3>").scale("C:minor")   // Change scales per cycle
n("0 2 4").scale("C:pentatonic")
n("0 2 4").scale("C4:major")      // With octave specification
n("0 2 4").scale("C:minor:pentatonic")  // Compound scales
```

### Transpose
```javascript
note("c e g").transpose(12)       // Transpose by semitones
note("c e g").transpose("<0 2 5>")// Pattern of transpositions
note("c e g").transpose("1P -2M 4P")  // Using intervals
```

### Scale Transpose (within scale)
```javascript
n("0 2 4").scale("C:major").scaleTranspose(1)
```

### Voicing
```javascript
chord("C Am F G").voicing()       // Turn chords into note voicings
n("0 1 2 3").chord("C Am F G").voicing()
```

### Common Chord Progressions (Scale Degrees)
Convert Roman numerals to scale degrees: I=0, ii=1, iii=2, IV=3, V=4, vi=5, vii=6

```javascript
// Pop (I-V-vi-IV)
n("0 4 5 3").scale("C:major").sound("piano")

// Jazz ii-V-I
n("1 4 0").scale("C:major").sound("piano")

// Blues 12-bar (I-I-I-I-IV-IV-I-I-V-IV-I-V)
n("0 0 0 0 3 3 0 0 4 3 0 4").scale("C:major").sound("piano")

// Folk (I-IV-I-V)
n("0 3 0 4").scale("C:major").sound("piano")

// Rock (I-bVII-IV-I) - use negative for flat
n("0 -1 3 0").scale("C:major").sound("sawtooth")

// Classical (I-IV-V-I)
n("0 3 4 0").scale("C:major").sound("piano")

// Modal (i-bVII-IV-i in minor)
n("0 -1 3 0").scale("C:minor").sound("sawtooth")

// EDM (i-VI-III-VII in minor)
n("0 5 2 6").scale("C:minor").sound("sawtooth")
```

---

## AUDIO EFFECTS

### Filter Effects

#### Low-Pass Filter (LPF)
```javascript
note("c2 c3").lpf(1000)           // Cutoff frequency
note("c2 c3").lpf("<200 1000 5000>")  // Pattern
note("c2 c3").lpf(1000).lpq(5)    // With resonance (q-value)
```

#### High-Pass Filter (HPF)
```javascript
s("bd hh").hpf(500)
s("bd hh").hpf(2000).hpq(10)
```

#### Band-Pass Filter (BPF)
```javascript
s("bd").bpf(1000)                 // Center frequency
s("bd").bpf(1000).bpq(5)          // With resonance
```

### Amplitude Envelope (ADSR)
```javascript
note("c3 e3 g3").attack(0.1).decay(0.2).sustain(0.5).release(0.3)
note("c3 e3 g3").adsr(".1:.2:.5:.3")  // Shorthand
```

### Gain & Volume
```javascript
sound("bd").gain(0.5)             // Volume (0-1)
sound("bd").gain("<0.2 1>")       // Pattern
sound("hh*8").gain("[.25 1]*4")   // Emphasis
sound("bd").velocity(0.8)         // Alternative to gain
sound("bd").postgain(1.5)         // After effects
```

### Tremolo (Amplitude Modulation)
```javascript
note("c3").tremsync(4)            // Speed in cycles
note("c3").tremsync(4).tremolodepth(0.5)
note("c3").tremsync(4).tremoloskew(0.5)
```

### Distortion
```javascript
note("c2").distort(5)             // Amount
note("c2").distort("8:.4")        // With postgain
note("c2").distort("3:0.5:diode") // With type
```

### Waveshaping
```javascript
sound("bd").coarse(4)             // Reduce sample rate
sound("bd").crush(8)              // Bit crush
sound("bd").shape(0.5)            // Waveshape
```

### Delay
```javascript
sound("bd").delay(0.5)            // Delay amount (0-1)
sound("bd").delay(0.5).delaytime(0.25)    // Delay time
sound("bd").delay(0.5).delayfeedback(0.7) // Feedback
sound("bd").delay(".5:.25:.9")    // All in one
```

### Reverb
```javascript
sound("bd").room(0.5)             // Reverb amount
sound("bd").room(0.8).roomsize(4) // Size
sound("bd").room(0.8).roomfade(2) // Fade time
sound("bd").room(0.8).roomlp(5000)// Lowpass
```

### Phaser
```javascript
note("c3").phaser(2)              // Speed Hz
note("c3").phaser(2).phaserdepth(0.75)
note("c3").phaser(2).phasercenter(1000)
note("c3").phaser(2).phasersweep(2000)
```

### Panning
```javascript
sound("bd").pan(0)                // Left (0) to right (1)
sound("bd").pan("<0 0.5 1>")      // Pattern
sound("bd").jux(rev)              // Strange stereo effect
sound("bd").juxBy(0.5, rev)       // Stereo width control
```

### Vowel Filter
```javascript
note("c3").vowel("<a e i o u>")   // Vowel sounds
```

### Ducker (Sidechain)
```javascript
s("bd").duckorbit(2).duckdepth(1).duckattack(0.2)
```

---

## TIME MODIFIERS

### Speed/Slow
```javascript
sound("bd hh").fast(2)            // Speed up (*2 equivalent)
sound("bd hh").slow(2)            // Slow down (/2 equivalent)
sound("bd hh").fast("<1 2 4>")    // Pattern of speeds
```

### Early/Late
```javascript
sound("bd").early(0.1)            // Start 0.1 cycles earlier
sound("bd").late(0.1)             // Start 0.1 cycles later
```

### Euclidean Rhythms (Functions)
```javascript
sound("bd").euclid(3, 8)          // 3 beats over 8 steps
sound("bd").euclidRot(3, 8, 2)    // With rotation
```

### Reverse & Palindrome
```javascript
sound("bd hh sd").rev()           // Reverse
sound("bd hh sd").palindrome()    // Forwards/backwards alternating
```

### Iteration
```javascript
note("0 1 2 3").iter(4)           // Shift by 1 each cycle
note("0 1 2 3").iterBack(4)       // Reverse iteration
```

### Repetition & Elongation
```javascript
sound("bd").ply(2)                // Repeat each event
sound("bd hh").ply("<1 2 3>")     // Pattern of repetitions
```

### Segment (Sample at rate)
```javascript
note(sine.range(40, 52)).segment(16)  // Sample at 16 segments/cycle
```

### Compress/Zoom
```javascript
sound("bd hh sd").compress(0.25, 0.75)  // Play portion in time
sound("bd hh sd").zoom(0.25, 0.75)      // Zoom into portion
```

### Linger
```javascript
sound("bd hh sd").linger(0.5)     // Loop first 50%
```

### Swing/Shuffle
```javascript
sound("hh*8").swing(4)            // Swing with 4 subdivisions
sound("hh*8").swingBy(0.33, 4)    // Custom swing amount
```

### Set Tempo (CPM)
```javascript
sound("bd").cpm(90)               // Cycles per minute
```

---

## RANDOM MODIFIERS

### Choose
```javascript
note("c d e").choose("piano", "sine")      // Random from list
sound("hh").choose("bd", "sd")             // Each event
note("c2 g2 d2").wchoose(["sine",10], ["triangle",1])  // Weighted
```

### ChooseCycles
```javascript
chooseCycles("bd", "hh", "sd").s()  // One per cycle
wchooseCycles(["bd",10], ["hh",1]).s()  // Weighted per cycle
```

### Degrade (Remove Events)
```javascript
sound("hh*8").degrade()           // 50% removal
sound("hh*8").degradeBy(0.2)      // 20% removal
sound("hh*8").undegradeBy(0.8)    // Keep only 80%
```

### Probability Functions
```javascript
sound("hh*8").sometimes(x => x.speed(0.5))           // 50%
sound("hh*8").often(x => x.speed(0.5))               // 75%
sound("hh*8").rarely(x => x.speed(0.5))              // 25%
sound("hh*8").almostAlways(x => x.speed(0.5))        // 90%
sound("hh*8").almostNever(x => x.speed(0.5))         // 10%
sound("hh*8").sometimesBy(0.3, x => x.speed(0.5))    // Custom %
sound("hh*8").someCycles(x => x.speed(0.5))          // Per cycle
```

### Random Utilities
```javascript
rand                              // Random 0-1
rand.range(0, 100)                // Random in range
rand.range(0, 100).segment(8)     // Multiple random values
sine, saw, square, tri            // Waveforms for modulation
perlin                            // Perlin noise
```

---

## TONAL FUNCTIONS

### Voicing (Chords to Notes)
```javascript
chord("C Am F G").voicing()
chord("C^7 A7 Dm7 G7").dict('ireal').voicing()
```

### Voicing Parameters
```javascript
n("0 1 2 3").chord("C Am F G")
  .voicing(mode: "below", anchor: "c4")
```

### Scale Degrees
```javascript
n("0 2 4 6").scale("C:major").note()  // Convert to notes
```

### Root Notes
```javascript
chord("C^7 Am Dm G").rootNotes(2)     // Get roots in octave 2
```

---

## SYNTHS & OSCILLATORS

### Basic Waveforms
```javascript
note("c3").sound("sine")          // Sine wave
note("c3").sound("triangle")      // Triangle
note("c3").sound("square")        // Square
note("c3").sound("sawtooth")      // Sawtooth
```

### Noise
```javascript
sound("white")                    // White noise
sound("pink")                     // Pink noise
sound("brown")                    // Brown noise
note("c3").noise(0.2)             // Add noise to oscillator
sound("crackle*4").density(0.1)   // Crackle noise
```

### FM Synthesis
```javascript
note("c3").fm(2)                  // FM brightness
note("c3").fm(4).fmh(1.5)         // Harmonicity ratio
note("c3").fm(4).fmattack(0.1).fmdecay(0.2).fmsustain(0.5)
```

### Additive Synthesis
```javascript
note("c3").partials([1, 0, 0.3, 0, 0.1])  // Harmonic magnitudes
note("c3").sound("user").partials([1, 0, 0.3, 0, 0.1])
```

### Vibrato
```javascript
note("c3").vib(4)                 // Vibrato frequency Hz
note("c3").vib(4).vibmod(2)       // Modulation depth semitones
```

### Wavetable Synthesis
```javascript
samples('bubo:waveforms')
note("g3").s('wt_flute')          // Any wt_ prefix = wavetable
```

### ZZFX (Zuper Zmall Zound Zynth)
```javascript
note("c2").s("z_sine").attack(0.01).decay(0.1).release(0.2)
note("c2").s("{z_sawtooth z_square}%4")
```

---

## SAMPLER EFFECTS

### Sample Playback Control
```javascript
sound("bd").begin(0.25)           // Skip first 25% of sample
sound("bd").end(0.75)             // Cut last 25%
sound("bd").loop(1)               // Loop the sample
sound("bd").loopBegin(0.25).loopEnd(0.75)  // Custom loop region
```

### Sample Manipulation
```javascript
sound("bd").cut(1)                // Cut group (like closed/open hh)
sound("bd").clip(0.5)             // Shorten duration
sound("bd").speed("<1 2 -1>")     // Change playback speed
sound("bd").speed("<1 1.5 2>")    // Pitch shift via speed
```

### Granular Effects
```javascript
sound("rhodes").chop(4)           // Granular synthesis
sound("breaks").striate(6)        // Trigger portions of sample
sound("breaks").slice(8, "0 1 2 3")  // Slice and trigger
sound("pad").splice(4, "0 1 2")   // Splice with speed matching
```

### Scrubbing
```javascript
sound("pad").scrub("0.5:2")       // Scrub with speed
sound("sample").scrub("{0.1!2 .25@3}%8")
```

### Looping
```javascript
sound("rhodes").loopAt(2)         // Fit sample into 2 cycles
sound("breaks").fit()             // Fit to event duration
```

### Advanced Loading
```javascript
// Load with pitch information
samples({
  moog: {
    'g3': 'moog/005_Mighty%20Moog%20G3.wav'
  }
}, 'github:tidalcycles/dirt-samples')

// Use Shabda for freesound samples
samples('shabda:bass:4,hihat:4')
```

---

## PATTERN FUNCTIONS

### Creating Patterns
```javascript
pattern([1, 2, 3, 4])             // From array
Pattern.of(1, 2, 3)               // Variadic
range(5)                          // 0-4
run(8)                            // 0-7
irand(10)                         // Random integer
```

### Pattern Combination
```javascript
pattern1.add(pattern2)            // Add values
pattern1.sub(pattern2)            // Subtract
pattern1.mul(pattern2)            // Multiply
pattern1.div(pattern2)            // Divide
```

### Mapping
```javascript
pattern.map(x => x * 2)           // Transform values
pattern.set(otherPattern)         // Set from pattern
```

### Cat & Stack
```javascript
cat(pattern1, pattern2)           // Concatenate in time
stack(pattern1, pattern2)         // Overlay in time
```

---

## TIPS & BEST PRACTICES

### Useful Shortcuts & Aliases
```javascript
s = sound
n = note
note = note
samples() = load samples

attack = a or att
decay = d or dec
sustain = sus or s
release = rel or r
lpf = cutoff, ctf, lp
hpf = hp, hcutoff
room = reverb
```

### Keyboard Shortcuts
```
Ctrl+Enter  = Play/update pattern
Ctrl+.      = Stop all sound
Ctrl+Z      = Undo
```

### Combining Parameters
```javascript
note("c e g").sound("piano")
  .lpf(800)
  .room(0.5)
  .delay(0.25)
  .gain(0.8)
```

### Working with Scales
```javascript
// Always use valid scale names:
"C:major", "A:minor", "D:dorian", "G:mixolydian"
"F:pentatonic", "Bb:blues"
```

### Tempo Guidelines
```
setcpm(30)  // Very slow (1 cycle = 2s)
setcpm(60)  // Medium
setcpm(120) // Fast
setcpm(240) // Very fast
```

### Orbits & Mixing
```javascript
sound("bd").orbit(1)              // Default orbit
sound("hh").orbit(2)              // Different reverb
s("bd").orbit("2,3")              // Multiple orbits (watch volume!)
```

### Common Patterns
```javascript
// Drum loop
sound("bd*4, [~ sd]*2, [hh]*8").bank("RolandTR909")

// Bassline
note("<[c2 c3]*4 [bb1 bb2]*4>").sound("sawtooth").lpf(800)

// Melody with scale
n("<0 2 4 [6 8]>").scale("C:minor").sound("piano")

// Full loop
$: sound("bd*4").bank("RolandTR909")
$: sound("[~ hh]*8")
$: note("<0 2 4 [6 8]>").scale("C:minor").sound("sawtooth")
```

### Modulation Patterns
```javascript
// LFO filter sweep
note("c3").lpf(sine.range(200, 2000).slow(4))

// Varying gain
sound("hh*16").gain(sine.range(0.3, 1).slow(2))

// Tremolo
sound("bd").tremsync(4).tremolodepth(0.5)
```

### Chaining Effects (Signal Order)
```
Sound → ADSR Envelope → Detune Effects
→ Gain → Filters (LP/HP/BP) → Vowel
→ Distortion → Tremolo → Compressor
→ Pan → Phaser → Delay (to orbit) → Reverb (to orbit)
```

---

## HANDY REFERENCES

### All Default Sound Names
```
Drums: bd, sd, rim, cp, hh, oh, cr, rd, ht, mt, lt, sh, cb, tb, perc, misc
Synths: sine, triangle, square, sawtooth
Noise: white, pink, brown, crackle
ZZFX: z_sine, z_square, z_sawtooth, z_tan, z_noise
Effects: fx
```

### Scale Types

For every key (C, C#, D, E, F, F#, G, A, B, and their respective sharps and flats), there are the following scales available:

```
major, minor, dorian, phrygian, lydian, mixolydian
pentatonic, blues, harmonic_minor
```

**Compound Scales:**
Scales can be combined using colon notation: `"C:minor:pentatonic"`, `"D:dorian:blues"`

**With Octaves:**
Scale notation can include octave: `"C4:major"`, `"A2:minor"`