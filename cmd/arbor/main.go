package main

import (
	"fmt"
	"os"

	"arbor/internal/cli"
)

func main() {
	if err := cli.Execute(os.Args[1:], cli.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
