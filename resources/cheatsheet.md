# STRUDEL QUICK REFERENCE

## BASIC SOUNDS
```
sound("bd")              // Kick drum
sound("hh")              // Hi-hat
sound("sd")              // Snare drum
sound("cp")              // Clap
sound("bd hh sd hh")     // Pattern sequence
```

## CORE FUNCTIONS
```
.fast(n)                 // Speed up by n
.slow(n)                 // Slow down by n
.gain(0-1)               // Volume (0 = silent, 1 = full)
.stack(pattern)          // Layer patterns
.cat(pattern)            // Concatenate patterns
```

## RHYTHM
```
sound("bd*4")            // Repeat 4 times
sound("bd [hh hh]")      // Subdivision
sound("bd hh*3 sd")      // Mix repeats
```

## MELODY
```
note("c a f e")          // Note sequence
note("c3 e3 g3")         // With octaves
.scale("C:minor")        // Set scale
```

## EFFECTS
```
.room(0-1)               // Reverb amount
.delay(0-1)              // Delay amount
.lpf(0-20000)            // Low-pass filter (frequency)
.hpf(0-20000)            // High-pass filter (frequency)
```

## COMMON PATTERNS
```
sound("bd").fast(4)                          // Four-on-the-floor kick
sound("hh").fast(8)                          // Fast hi-hats
sound("bd").stack(sound("hh").fast(2))       // Layered drums
note("c e g").slow(2)                        // Slow arpeggio
sound("bd sd").gain(0.8).room(0.3)           // Drums with reverb
```
