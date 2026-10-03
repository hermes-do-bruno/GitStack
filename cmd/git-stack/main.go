package main

import (
	"fmt"
	"os"

	"git-cascade/internal/gitstack"
)

func main() {
	app := gitstack.NewApp()
	if err := app.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(app.Stderr, err)
		os.Exit(1)
	}
}
