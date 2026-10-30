package main

import (
	"os"

	"github.com/pborenstein/gh-repopotamus/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
