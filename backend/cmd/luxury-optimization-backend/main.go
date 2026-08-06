package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GofMan5/Luxury-Optimization/internal/app"
	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(optimizer.RunCLI(os.Args[1:]))
	}
	go optimizer.RunAutoUpdate()
	if err := app.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "Luxury Optimization backend:", err)
		os.Exit(1)
	}
}
