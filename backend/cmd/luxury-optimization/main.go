package main

import (
	"os"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func main() {
	os.Exit(optimizer.RunCLI(os.Args[1:]))
}
