# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `perennial-tool`, a Go utility library for managing opam files, specifically designed to handle pin-depends and their transitive indirect dependencies. The tool is used in the context of the Perennial verification framework.

## Build and Test Commands

```bash
# Run tests with verbose output
go test ./... -v

# Run a specific test
go test ./... -run TestParse

# Run a specific test in a specific file
go test -run TestParse ./opam/opam_file_test.go ./opam/opam_file.go
```

## Architecture

### Package Structure

The codebase consists of a single `opam` package with two main components:

1. **opam_file.go** - Opam file parsing and manipulation
2. **update.go** - Git repository interaction and dependency resolution

### Core Data Structures

**`region` struct**: Represents a half-open interval `[startLine, endLine)` in the opam file. Used to track:
- `depends`: The depends block
- `pinDepends`: The pin-depends block (includes opening `[` and closing `]`)
- `indirectPinDepends`: The indirect dependencies section (delimited by `## begin indirect` and `## end` markers)

The `region` struct uses an **exclusive endLine** convention, making it compatible with Go's slice operations.

**`OpamFile` struct**: Contains the parsed opam file as a slice of lines plus region metadata. All modifications to the file work by manipulating the `Lines` slice using Go's `slices` package functions (`Insert`, `Delete`, `Replace`).

**`PinDepend` struct**: Represents a single pin-depends entry with:
- `Package`: Package name (without `.dev` suffix when stored)
- `URL`: Git URL (without `git+` prefix and without commit hash)
- `Commit`: Commit hash (normalized to 15 characters)

### Key Design Patterns

1. **Line-based editing**: The opam file is kept as `[]string` (lines), and all edits use `slices.Insert`, `slices.Delete`, and `slices.Replace`. After modifications, `findRegions()` is called to update region boundaries.

2. **Region tracking**: The parser identifies three regions (depends, pinDepends, indirectPinDepends) and updates them after any structural changes to the file.

3. **Normalization**:
   - Package names: Stored without `.dev` suffix, added on formatting
   - Commit hashes: Normalized to first 15 characters
   - Git URLs: Stored without `git+` prefix internally

4. **Indirect dependencies**: The tool distinguishes between direct pin-depends and indirect (transitive) dependencies. Indirect deps are marked with `## begin indirect` / `## end` comments in the opam file.

### Testing Notes

Run the tests after making changes and make sure they pass.

- Tests use `github.com/stretchr/testify` for assertions
- `update_test.go` contains live tests that make actual network/git calls (marked with comments)
- Line numbers in test assertions are 0-indexed and refer to the `exampleOpam` constant
- The test example file uses a specific format with indirect dependencies for perennial and its transitive deps
