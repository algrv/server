package agent

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	cheatsheetCache string
	cheatsheetOnce  sync.Once
)

func getCheatsheet() string {
	cheatsheetOnce.Do(func() {
		content, err := os.ReadFile(filepath.Join("resources", "cheatsheet.md"))

		if err != nil {
			cheatsheetCache = ""
			return
		}

		cheatsheetCache = string(content)
	})

	return cheatsheetCache
}
