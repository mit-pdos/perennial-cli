# perennial-cli

Tool to manage perennial-based verification projects.

## Features

Run `perennial-cli help` for more detailed help.

### Manage opam files

Perennial projects use an opam file to specify their dependencies, notably the version of perennial. These dependencies are specified using a git hash using [opam's pin-depends feature](https://opam.ocaml.org/doc/Manual.html#opamfield-pin-depends). While pin-depends allows depending on specific commits and removes the need for a custom opam repository, it has some quirks: `opam upgrade` does not update pin-depends, and opam does not install transitive pin-depends for a dependency.

We handle this by proving `perennial-cli opam update`, which can (a) update the pin-depends field to the latest commit, and (b) automatically maintain all indirect dependencies.

An easy feature to add would be `perennial-cli opam add` to add a dependency by URL.

### Run goose

`perennial-cli goose` will run goose. Write a `goose.toml` file to configure the translation:

```toml
rocq = "src"
go_path = "."
packages = ["./..."]
```

All of these fields are optional; an empty file is enough to direct `perennial-cli`. If `go_path` is not specified, the default behavior is to search for `go.mod`, so your code can be in a subdirectory.

> [!NOTE]
> This functionality should be integrated into the `goose` binary.

### Install and uninstall files

`perennial-cli install` implements the functionality of `make install` when using `rocq makefile`. It has some extra features: it takes a list of files to install and uses `.rocqdeps.d` (generated as part of our Makefile setup) to automatically extend that list with all dependencies.

`perennial-cli uninstall` does the same as `make uninstall`.

This command is intended to be called by the opam file, but it can be run manually (with the caveat that the installed files may not match what opam thinks is installed).

## Developing

perennial-cli requires Go 1.24+.
