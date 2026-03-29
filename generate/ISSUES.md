# generate/ package issues

Sorted by priority (bugs first, then logic, then quality).

## Bugs (must fix before merge)

1. **Anchor collision in llmstxt.go** — Uses `schema.Name` for markdown anchors instead of `CommandPath`. Two commands named "add" (`db add`, `user add`) produce duplicate anchors, breaking links. (`llmstxt.go:78`)

2. **YAML special character escaping in skill.go** — YAML frontmatter doesn't quote values containing `:`, `#`, or other special chars. `Author: "Alice: expert"` produces invalid YAML. (`skill.go:48-67`)

## Logic inconsistencies (should fix)

3. **Leaf command detection differs across generators** — skill.go uses OR (`no subcommands OR has flags`), llmstxt.go uses AND (`no subcommands AND (has flags OR has RunE)`), agents.go includes all commands. A command with both subcommands AND flags is treated differently by each. (`skill.go:221`, `llmstxt.go:68`)

4. **Required flags not marked in llms.txt** — agents.go and skill.go indicate required flags; llmstxt.go shows `(type, default)` with no required indicator. An agent can't tell which flags are mandatory. (`llmstxt.go:95-112`)

5. **Empty description handling inconsistent** — agents.go uses `"-"`, llmstxt.go uses command name, skill.go omits entirely. Should pick one strategy. (`agents.go:59`, `llmstxt.go:80`, `skill.go:124`)

6. **Empty flag description handling inconsistent** — agents.go uses `"-"`, skill.go and llmstxt.go output empty string. (`agents.go:80-82`, `skill.go:163`, `llmstxt.go:110`)

7. **Env var deduplication inconsistent** — agents.go deduplicates across commands; skill.go and llmstxt.go don't. Shared env vars appear multiple times. (`agents.go:127-142`)

8. **Command sorting inconsistent** — agents.go and llmstxt.go sort by CommandPath; skill.go doesn't sort. Non-deterministic output from skill.go. (`skill.go:75`)

9. **Mixed subcommand+flags edge case** — A command that has BOTH subcommands AND flags is a leaf in skill.go (OR logic) but not in llmstxt.go (AND logic). Different generators produce different command lists for the same CLI.

10. **Agents.go missing empty root description guard** — If root description is empty, agents.go outputs blank content unconditionally. llmstxt.go correctly guards with `if != ""`. (`agents.go:43-44`)

## Code quality (should fix)

11. **Duplicate kebab-case functions** — `toKebabCase()` in skill.go and `toAnchor()` in llmstxt.go are identical. Deduplicate to generate.go. (`skill.go:267-271`, `llmstxt.go:159-163`)

12. **Dead code in skill.go** — `if def == "" { def = "" }` is a no-op. (`skill.go:159-162`)

13. **Redundant flagNames sorting in llmstxt.go** — Same slice built and sorted twice in the same loop iteration. Build once, reuse. (`llmstxt.go:98-102`, `llmstxt.go:119-123`)

14. **Hardcoded "config" flag lookup in agents.go** — Checks `rootCmd.Flags().Lookup("config")` by name. CLIs using `--conf` or `--config-file` are missed. Should use the config flag annotation. (`agents.go:106`)

15. **Misleading "per-command schema" comment** — agents.go says `--jsonschema` gives "per-command schema" but it gives the current command's schema. (`agents.go:112`)

16. **Description truncation limit undocumented** — 1024 char limit in skill.go comes from Anthropic's skill spec but isn't cited. Should reference the spec or define a constant. (`skill.go:240-242`)

## Missing tests

17. **No test for commands with no flags** — All tests use buildTestTree() with flags on every command.

18. **No test for empty descriptions** — All test commands have Short set.

19. **No test for nil/empty schema.Flags** — Only tested with populated flags.

20. **No test for env var deduplication across commands** — Missing scenario where two commands bind the same env var.

21. **No test for duplicate anchor collision** — No test verifying unique anchors in llmstxt output.

22. **No test for YAML special characters in metadata** — No test for Author containing `:` or `#`.

## Low priority / documentation

23. **Options types inconsistent** — SkillOptions has Author/Version/MCPServer; LLMsTxtOptions and AgentsOptions have ModulePath. Different fields for different formats. Intentional but worth documenting.

24. **Required flags cell overflow risk in agents.go** — Many required flags produce a very long table cell. No truncation. (`agents.go:145-155`)

25. **No input validation** — No checks that CommandPath is valid, flags don't conflict, root command exists. Silently produces malformed output if input is bad.

---

## Follow-up tasks

26. **Document what humans should add on top of generated files** — The generators produce mechanically correct scaffolds (flags, types, defaults, env vars). Document clearly that humans should add: (a) trigger phrases for skill discovery, (b) workflow guidance and step-by-step instructions, (c) realistic examples with domain-specific values, (d) error handling advice and troubleshooting sections, (e) negative triggers ("do NOT use for..."). This should be in both the generate package godoc and a section in the README.

27. **Dogfood: generate discovery files for examples/full** — Add a `go:generate` directive to `examples/full` that runs the three generators on its command tree. Commit the output (SKILL.md, llms.txt, AGENTS.md) as living proof that the generators work on a real structcli CLI. This also serves as documentation-by-example for users.
