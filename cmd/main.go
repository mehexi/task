package main

import (
	"fmt"
	"os"

	"github.com/mehexi/task/internal"
)

func main() {
	if err := internal.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
