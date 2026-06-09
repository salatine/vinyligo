package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
)

const MLCharacterLimit = 60

func EditTitle(title string) string {
	if runtime.GOOS == "windows" {
		return editTitleWindows(title)
	}
	return editTitleReadline(title)
}

func editTitleReadline(title string) string {
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

func editTitleWindows(title string) string {
	fmt.Printf("\ntítulo excede %d chars (%d): %s\n", MLCharacterLimit, len(title), title)

	tmpFile, err := os.CreateTemp("", "vinyligo-title-*.txt")
	if err != nil {
		return title[:MLCharacterLimit]
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	tmpFile.WriteString(title)
	tmpFile.Close()

	cmd := exec.Command("notepad.exe", tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return title[:MLCharacterLimit]
	}

	newTitle := strings.TrimSpace(string(data))
	if newTitle == "" || len(newTitle) > MLCharacterLimit {
		fmt.Printf("título inválido (%d chars), truncando\n", len(newTitle))
		if newTitle == "" {
			return title[:MLCharacterLimit]
		}
		return newTitle[:MLCharacterLimit]
	}
	return newTitle
}
