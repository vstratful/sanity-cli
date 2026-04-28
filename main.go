package main

import (
	"fmt"
	"os"

	"github.com/vstratful/sanity-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if !cmd.IsQuiet(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
