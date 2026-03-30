// Command generate produces SKILL.md, llms.txt, and AGENTS.md for the full example CLI.
//
// Usage:
//
//	go run ./cmd/generate
//
// Files are written to the examples/full/ directory.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	full_example_cli "github.com/leodido/structcli/examples/full/cli"
	"github.com/leodido/structcli/generate"
)

func main() {
	log.SetFlags(0)

	// Build the command tree (same as main.go, but don't execute)
	rootCmd, err := full_example_cli.NewRootC(false)
	if err != nil {
		log.Fatalf("building CLI: %v", err)
	}

	// Resolve output directory to examples/full/ (where this tool lives)
	_, thisFile, _, _ := runtime.Caller(0)
	outDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	modulePath := "github.com/leodido/structcli/examples/full"

	// Generate SKILL.md
	skillBytes, err := generate.Skill(rootCmd, generate.SkillOptions{
		Author:  "leodido",
		Version: "0.9.0",
	})
	if err != nil {
		log.Fatalf("generating SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "SKILL.md"), skillBytes, 0644); err != nil {
		log.Fatalf("writing SKILL.md: %v", err)
	}
	log.Printf("wrote %s", filepath.Join(outDir, "SKILL.md"))

	// Generate llms.txt
	llmsBytes, err := generate.LLMsTxt(rootCmd, generate.LLMsTxtOptions{
		ModulePath: modulePath,
	})
	if err != nil {
		log.Fatalf("generating llms.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "llms.txt"), llmsBytes, 0644); err != nil {
		log.Fatalf("writing llms.txt: %v", err)
	}
	log.Printf("wrote %s", filepath.Join(outDir, "llms.txt"))

	// Generate AGENTS.md
	agentsBytes, err := generate.Agents(rootCmd, generate.AgentsOptions{
		ModulePath: modulePath,
	})
	if err != nil {
		log.Fatalf("generating AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "AGENTS.md"), agentsBytes, 0644); err != nil {
		log.Fatalf("writing AGENTS.md: %v", err)
	}
	log.Printf("wrote %s", filepath.Join(outDir, "AGENTS.md"))
}
