package utils

import (
	"fmt"

	"github.com/chzyer/readline"
)

const MLCharacterLimit = 60

func EditTitle(title string) string {
	fmt.Printf("\ntítulo excede %d chars (%d): %s\n", MLCharacterLimit, len(title), title)

	rl, err := readline.NewEx(&readline.Config{Prompt: "novo título: "})
	if err != nil {
		return title[:MLCharacterLimit]
	}
	defer rl.Close()

	for {
		line, err := rl.ReadlineWithDefault(title)
		if err != nil {
			return title[:MLCharacterLimit]
		}
		if len(line) <= MLCharacterLimit {
			return line
		}
		fmt.Printf("ainda excede (%d/%d)\n", len(line), MLCharacterLimit)
	}
}
