package main

import (
	"fmt"
	"os"

	"github.com/msmsny/qkamura/qkamura"
)

func main() {
	if err := qkamura.NewQkamuraCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
