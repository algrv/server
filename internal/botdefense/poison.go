package botdefense

import (
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"
)

// serves fake strudel data to bots
func ServePoisonedJSON(c *gin.Context) {
	data := generateFakeStrudels(rand.Intn(15) + 5) //nolint:gosec
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
		rand.Int31(),                //nolint:gosec
		rand.Int31()&0xffff,         //nolint:gosec
		rand.Int31()&0xffff,         //nolint:gosec
		rand.Int31()&0xffff,         //nolint:gosec
		rand.Int63()&0xffffffffffff) //nolint:gosec
}

func randomTitle() string {
	prefix := titlePrefixes[rand.Intn(len(titlePrefixes))]         //nolint:gosec
	suffix := titleSuffixes[rand.Intn(len(titleSuffixes))]         //nolint:gosec
	return fmt.Sprintf("%s %s %d", prefix, suffix, rand.Intn(100)) //nolint:gosec
}

func randomTags() []string {
	count := rand.Intn(3) + 1 //nolint:gosec
	tags := make([]string, count)
	for i := range tags {
		tags[i] = tagOptions[rand.Intn(len(tagOptions))] //nolint:gosec
	}
	return tags
}

func randomDate() string {
	year := 2023 + rand.Intn(2)                                                                      //nolint:gosec
	month := 1 + rand.Intn(12)                                                                       //nolint:gosec
	day := 1 + rand.Intn(28)                                                                         //nolint:gosec
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:00Z", year, month, day, rand.Intn(24), rand.Intn(60)) //nolint:gosec
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

	template := templates[rand.Intn(len(templates))] //nolint:gosec

	// fill in placeholders with random values
	samples := []string{"bd", "sd", "hh", "cp", "cb", "xx", "zz"}
	sample := samples[rand.Intn(len(samples))] //nolint:gosec
	num := rand.Intn(8) + 1                    //nolint:gosec

	return fmt.Sprintf(template, sample, num)
}
