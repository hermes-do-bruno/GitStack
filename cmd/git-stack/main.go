package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hermes-do-bruno/GitStack/internal/gitstack"
)

func main() {
	app := gitstack.NewApp()
	if err := app.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(app.Stderr, formatError(err.Error()))
		os.Exit(1)
	}
}

func formatError(msg string) string {
	lines := strings.Split(strings.TrimSuffix(msg, "\n"), "\n")
	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, "git-stack: ") {
			continue
		}
		lines[i] = "git-stack: " + line
	}
	return strings.Join(lines, "\n")
}
