// Command generate produces SKILL.md, llms.txt, and AGENTS.md for the full example CLI.
//
// Usage:
//
//	go run ./cmd/generate
//
// Invoked via the //go:generate directive in main.go, files are written to the
// package directory (examples/full/) because go generate sets the working directory
// to the package containing the directive.
package main

import (
	"log"
	"os"

	full_example_cli "github.com/leodido/structcli/examples/full/cli"
	"github.com/leodido/structcli/generate"
)

func main() {
	log.SetFlags(0)

	rootCmd, err := full_example_cli.NewRootC(false)
	if err != nil {
		log.Fatalf("building CLI: %v", err)
	}

	// go generate sets cwd to the package directory (examples/full/), so files land there.
	outDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("resolving output directory: %v", err)
	}

	if err := generate.WriteAll(rootCmd, outDir, generate.AllOptions{
		ModulePath: "github.com/leodido/structcli/examples/full",
		Skill: generate.SkillOptions{
			Author:  "leodido",
			Version: "0.9.0",
		},
	}); err != nil {
		log.Fatal(err)
	}
}
