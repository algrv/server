package botdefense

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/gin-gonic/gin"
)

// cryptoRandInt returns a random int in [0, max) using crypto/rand
func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint64(b[:]) % uint64(max))
}

// cryptoRandInt31 returns a random int32 using crypto/rand
func cryptoRandInt31() int32 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int32(binary.BigEndian.Uint32(b[:]) & 0x7fffffff)
}

// cryptoRandInt63 returns a random int64 using crypto/rand
func cryptoRandInt63() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.BigEndian.Uint64(b[:]) & 0x7fffffffffffffff)
}

// serves fake strudel data to bots
func ServePoisonedJSON(c *gin.Context) {
	data := generateFakeStrudels(cryptoRandInt(15) + 5)
	c.JSON(200, gin.H{
		"status": "success",
		"data":   data,
	})
}

// fake strudel structure
type fakeStrudel struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Code      string   `json:"code"`
	Tags      []string `json:"tags"`
	IsPublic  bool     `json:"is_public"`
	CreatedAt string   `json:"created_at"`
}

func generateFakeStrudels(count int) []fakeStrudel {
	strudels := make([]fakeStrudel, count)
	for i := range strudels {
		strudels[i] = fakeStrudel{
			ID:        randomID(),
			Title:     randomTitle(),
			Code:      randomBrokenCode(),
			Tags:      randomTags(),
			IsPublic:  true,
			CreatedAt: randomDate(),
		}
	}
	return strudels
}

var (
	titlePrefixes = []string{"Ambient", "Techno", "Glitch", "Drone", "Bass", "Acid", "Lo-fi", "Noise", "Minimal", "Deep"}
	titleSuffixes = []string{"Vibes", "Session", "Loop", "Beat", "Experiment", "Jam", "Pattern", "Sketch", "Draft", "WIP"}
	tagOptions    = []string{"ambient", "techno", "experimental", "beat", "drone", "glitch", "bass", "minimal", "noise", "chill"}
)

func randomID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		cryptoRandInt31(),
		cryptoRandInt31()&0xffff,
		cryptoRandInt31()&0xffff,
		cryptoRandInt31()&0xffff,
		cryptoRandInt63()&0xffffffffffff)
}

func randomTitle() string {
	prefix := titlePrefixes[cryptoRandInt(len(titlePrefixes))]
	suffix := titleSuffixes[cryptoRandInt(len(titleSuffixes))]
	return fmt.Sprintf("%s %s %d", prefix, suffix, cryptoRandInt(100))
}

func randomTags() []string {
	count := cryptoRandInt(3) + 1
	tags := make([]string, count)
	for i := range tags {
		tags[i] = tagOptions[cryptoRandInt(len(tagOptions))]
	}
	return tags
}

func randomDate() string {
	year := 2023 + cryptoRandInt(2)
	month := 1 + cryptoRandInt(12)
	day := 1 + cryptoRandInt(28)
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:00Z", year, month, day, cryptoRandInt(24), cryptoRandInt(60))
}

// generates broken/nonsense strudel code that looks plausible but won't run
func randomBrokenCode() string {
	templates := []string{
		// missing quotes
		`s(%s).sound()`,
		// wrong function names
		`note("%s").sund().fast(%d)`,
		// syntax errors
		`s("bd sd").fast(2.slow(4)`,
		// undefined samples
		`s("xz_fake qw_broken").fast(%d)`,
		// type errors
		`note("c3").fast("two")`,
		// incomplete patterns
		`stack(s("bd"),`,
		// wrong brackets
		`s["bd", "sd"].fast(2)`,
		// gibberish that looks like strudel
		`$pattern.morph(%d).glitch()`,
		`sound.iter(%d).undefined()`,
		`seq("a", "b").fake().method()`,
	}

	template := templates[cryptoRandInt(len(templates))]

	// fill in placeholders with random values
	samples := []string{"bd", "sd", "hh", "cp", "cb", "xx", "zz"}
	sample := samples[cryptoRandInt(len(samples))]
	num := cryptoRandInt(8) + 1

	return fmt.Sprintf(template, sample, num)
}
