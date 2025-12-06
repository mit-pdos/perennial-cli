# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`perennial-cli` is a Go CLI tool for managing perennial projects. It has several orthogonal features: managing opam files, analyzing Makefile dependencies, creating a new perennial project, and installing compiled outputs.

## Test Commands

Use `go test` to test any changes.

## Packages

**cmd** has the CLI implementation, split into files for each subcommand. The CLI uses cobra.

The other packages implement the main functionalities of the CLI, exported as libraries so they can be tested individually:

- **opam** implements support for parsing and updating opam files (specifically depends and pin-depends)
- **git** interacts with git remotes
- **init_proj** creates a new Go project
- **depgraph** analyzes dependencies from `rocq dep`
- **rocq_makefile** extracts info from `rocq makefile`
- **goose_proj** parses `goose.toml` files
