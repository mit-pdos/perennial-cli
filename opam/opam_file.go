package opam

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"regexp"
	"slices"
	"strings"
)

const HASH_ABBREV_LENGTH = 10

// Abbreviate commit hash
func AbbreviateHash(commit string) string {
	if len(commit) > HASH_ABBREV_LENGTH {
		return commit[:HASH_ABBREV_LENGTH]
	}
	return commit
}

var (
	// Regex patterns for parsing opam files
	dependsRe       = regexp.MustCompile(`^\s*depends:\s*\[`)
	pinDependsRe    = regexp.MustCompile(`^\s*pin-depends:\s*\[`)
	closeBracketRe  = regexp.MustCompile(`^\s*\]`)
	beginIndirectRe = regexp.MustCompile(`^\s*##\s*begin indirect\b.*$`)
	endIndirectRe   = regexp.MustCompile(`^\s*##\s*end\b.*$`)
	// Matches: ["package.name" "git+https://...#commit"]
	pinDependLineRe = regexp.MustCompile(`^\s*\[\s*"([^"]+)"\s+"([^"]+)"\s*\]`)
	// Matches dependency lines: "package-name" or "package-name" { version-constraint }
	dependLineRe = regexp.MustCompile(`^\s*"([^"]+)"`)
)

type PinDepend struct {
	Package string // package name (e.g., rocq-iris)
	URL     string // URL (with git+https protocol)
	Commit  string // commit hash
}

// Normalize fixes dep.
//
// Returns dep.
func (dep *PinDepend) Normalize() *PinDepend {
	dep.Package = strings.TrimSuffix(dep.Package, ".dev")
	if strings.HasPrefix("https://", dep.URL) {
		dep.URL = "git+" + dep.URL
	}
	dep.Commit = AbbreviateHash(dep.Commit)
	return dep
}

type region struct {
	startLine int
	endLine   int // exclusive
}

func (r region) Contains(line int) bool {
	return r.startLine <= line && line < r.endLine
}

func (r region) empty() bool {
	return r.endLine <= r.startLine
}

func rangeIter(start, end int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := start; i < end; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func (r region) innerLineNums() iter.Seq[int] {
	return rangeIter(r.startLine+1, r.endLine-1)
}

type OpamFile struct {
	Lines []string
	// depends defines the region with the depends: block.
	depends region
	// pinDepends defines the start and end of the pin-depends: block.
	//
	// The region includes the pin-depends: [ and ] lines.
	pinDepends region
	// indirectPinDepends defines the start and end of the region with the
	// indirect dependencies, delimited by ## begin indirect and ## end markers.
	// This will be a sub-range of pinDepends.
	indirectPinDepends region
}

// findRegions parses the depends and pinDepends sections from f.Lines
func (f *OpamFile) findRegions() error {
	f.depends = region{}
	f.pinDepends = region{}
	f.indirectPinDepends = region{}

	inDepends := false
	inPinDepends := false
	indirectStart := -1

	for i, line := range f.Lines {
		// Check for depends: [ block
		if !inDepends && dependsRe.MatchString(line) {
			f.depends.startLine = i
			inDepends = true
			continue
		}

		// Check for pin-depends: [ block
		if !inPinDepends && pinDependsRe.MatchString(line) {
			f.pinDepends.startLine = i
			inPinDepends = true
			continue
		}

		// Check for closing ] of depends
		if inDepends && closeBracketRe.MatchString(line) {
			f.depends.endLine = i + 1
			inDepends = false
			continue
		}

		// Check for closing ] of pin-depends
		if inPinDepends && closeBracketRe.MatchString(line) {
			f.pinDepends.endLine = i + 1
			inPinDepends = false

			// Check for unclosed indirect region
			if indirectStart >= 0 && f.indirectPinDepends.empty() {
				return fmt.Errorf("unclosed indirect region starting at line %d", indirectStart)
			}
			continue
		}

		// Check for indirect dependency markers within pin-depends
		if inPinDepends {
			if beginIndirectRe.MatchString(line) {
				if indirectStart >= 0 {
					return fmt.Errorf("nested ## begin indirect markers at lines %d and %d", indirectStart, i)
				}
				indirectStart = i
			} else if endIndirectRe.MatchString(line) {
				if indirectStart < 0 {
					return fmt.Errorf("## end marker without ## begin indirect at line %d", i)
				}
				f.indirectPinDepends.startLine = indirectStart
				f.indirectPinDepends.endLine = i + 1
				indirectStart = -1
			}
		}
	}

	// Check for unclosed blocks
	if inDepends {
		return fmt.Errorf("unclosed depends block starting at line %d", f.depends.startLine)
	}
	if inPinDepends {
		return fmt.Errorf("unclosed pin-depends block starting at line %d", f.pinDepends.startLine)
	}

	return nil
}

// update parsed data after changing f.Lines
//
// Internal function: errors cause a panic() since this library should not
// introduce parse errors
func (f *OpamFile) update() {
	if err := f.findRegions(); err != nil {
		panic(fmt.Errorf("internal error: %w", err))
	}
}

func Parse(r io.Reader) (*OpamFile, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	f := &OpamFile{Lines: lines}
	err := f.findRegions()
	if err != nil {
		return nil, err
	}
	if f.depends.empty() {
		f.Lines = slices.Insert(f.Lines, f.depends.endLine, "depends: [", "]")
		f.depends = region{startLine: len(f.Lines) - 2, endLine: len(f.Lines)}
	}
	if f.pinDepends.empty() {
		f.Lines = slices.Insert(f.Lines, f.depends.endLine, "pin-depends: [", "]")
		f.update()
	}
	return f, nil
}

// String returns the opam file as a string
func (f *OpamFile) String() string {
	return strings.Join(f.Lines, "\n") + "\n"
}

// parsePinDependLine parses a line like:
//
//	["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbdea"]
func parsePinDependLine(line string) *PinDepend {
	matches := pinDependLineRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	packageName := matches[1]

	fullURL := matches[2]

	// Split URL into base and commit (split on #)
	url := fullURL
	commit := ""
	if idx := strings.IndexByte(fullURL, '#'); idx >= 0 {
		url = fullURL[:idx]
		commit = fullURL[idx+1:]
	}

	dep := &PinDepend{
		Package: packageName,
		URL:     url,
		Commit:  commit,
	}
	return dep.Normalize()
}

// String formats a PinDepend as an opam pin-depends line
func (dep PinDepend) String() string {
	fullURL := dep.URL
	if dep.Commit != "" {
		fullURL = dep.URL + "#" + AbbreviateHash(dep.Commit)
	}
	fullPackageName := dep.Package + ".dev"
	// Use spacing similar to the example: package name padded with spaces between quotes
	// Total width is package name in quotes (package + 2 for quotes) padded to 27 chars
	return fmt.Sprintf("  [%-27s \"%s\"]", "\""+fullPackageName+"\"", fullURL)
}

// GetPinDepends returns all direct pin-depends (excluding indirect dependencies).
func (f *OpamFile) GetPinDepends() []PinDepend {
	var deps []PinDepend
	for i := range f.pinDepends.innerLineNums() {
		// Skip lines in the indirect section
		if f.indirectPinDepends.Contains(i) {
			continue
		}

		line := f.Lines[i]
		dep := parsePinDependLine(line)
		if dep != nil {
			deps = append(deps, *dep)
		}
	}

	return deps
}

// AddPinDepend adds or updates a pin-depends entry in the opam file.
// If an entry for the package already exists, it will be replaced.
// If the package is in the indirect section, it will be removed from there.
// If no pin-depends block exists in the file, the function returns without changes.
// The new entry is added immediately after the "pin-depends: [" line if it doesn't already exist.
func (f *OpamFile) AddPinDepend(dep PinDepend) {
	if f.pinDepends.empty() {
		return
	}

	// Search for existing entry and replace it
	foundIndex := -1
	for i := range f.pinDepends.innerLineNums() {
		existingDep := parsePinDependLine(f.Lines[i])
		if existingDep != nil && existingDep.Package == dep.Package {
			foundIndex = i
			break
		}
	}

	// If found in indirect section, remove it from there and add to main section
	if f.indirectPinDepends.Contains(foundIndex) {
		// Remove from indirect section
		f.Lines = slices.Delete(f.Lines, foundIndex, foundIndex+1)

		f.update()

		// Add to main section (after pin-depends: [ line)
		f.Lines = slices.Insert(f.Lines, f.pinDepends.startLine+1, dep.String())
	} else if foundIndex >= 0 {
		// Found in main section, just replace it
		f.Lines[foundIndex] = dep.String()
	} else {
		// Not found anywhere, add it after the pin-depends: [ line
		f.Lines = slices.Insert(f.Lines, f.pinDepends.startLine+1, dep.String())
	}

	f.update()
}

func (f *OpamFile) GetIndirect() []PinDepend {
	if f.indirectPinDepends.empty() {
		return nil
	}

	var deps []PinDepend
	start := f.indirectPinDepends.startLine + 1 // Skip "## begin indirect" line
	end := f.indirectPinDepends.endLine - 1     // Skip "## end" line

	for i := start; i < end; i++ {
		line := f.Lines[i]
		dep := parsePinDependLine(line)
		if dep != nil {
			deps = append(deps, *dep)
		}
	}

	return deps
}

func (f *OpamFile) SetIndirect(indirects []PinDepend) {
	if f.pinDepends.empty() {
		return
	}

	// First, update any packages that are already in the main pin-depends section
	// and filter them out from the indirects list
	var filteredIndirects []PinDepend
	for _, indirect := range indirects {
		found := false
		start := f.pinDepends.startLine + 1

		// Check if package exists in main pin-depends (outside indirect section)
		for i := start; i < f.pinDepends.endLine-1; i++ {
			// Skip lines in indirect section
			if f.indirectPinDepends.Contains(i) {
				continue
			}

			existingDep := parsePinDependLine(f.Lines[i])
			if existingDep != nil && existingDep.Package == indirect.Package {
				// Update the existing entry
				f.Lines[i] = indirect.String()
				found = true
				break
			}
		}

		// Only add to indirect section if not found in main section
		if !found {
			filteredIndirects = append(filteredIndirects, indirect)
		}
	}

	// If there's already an indirect region, replace it
	if !f.indirectPinDepends.empty() {
		// Build new indirect section
		indirectLines := []string{"  ## begin indirect"}
		for _, dep := range filteredIndirects {
			indirectLines = append(indirectLines, dep.String())
		}
		indirectLines = append(indirectLines, "  ## end")

		// Replace the indirect region
		start := f.indirectPinDepends.startLine
		end := f.indirectPinDepends.endLine

		f.Lines = slices.Replace(f.Lines, start, end, indirectLines...)
	} else {
		// Add new indirect section before the closing ] of pin-depends
		indirectLines := []string{
			"",
			"  ## begin indirect",
		}
		for _, dep := range filteredIndirects {
			indirectLines = append(indirectLines, dep.String())
		}
		indirectLines = append(indirectLines, "  ## end")

		// Insert before the closing ] of pin-depends
		insertPos := f.pinDepends.endLine - 1

		f.Lines = slices.Insert(f.Lines, insertPos, indirectLines...)
	}
	f.update()
}

// GetDependencies returns all dependencies listed in the depends block,
// ignoring version constraints. Returns just the package names.
func (f *OpamFile) GetDependencies() []string {
	if f.depends.empty() {
		return nil
	}

	var deps []string
	for i := range f.depends.innerLineNums() {
		line := f.Lines[i]
		matches := dependLineRe.FindStringSubmatch(line)
		if matches != nil {
			deps = append(deps, matches[1])
		}
	}

	return deps
}

// AddDependency adds a new dependency to the depends block.
// If the dependency already exists, it does nothing.
// The dependency is added without version constraints.
func (f *OpamFile) AddDependency(packageName string) {
	if f.depends.empty() {
		return
	}

	// Check if dependency already exists
	existingDeps := f.GetDependencies()
	for _, dep := range existingDeps {
		if dep == packageName {
			return // Already exists, nothing to do
		}
	}

	// Add the new dependency after the opening "depends: [" line
	newLine := fmt.Sprintf("  \"%s\"", packageName)
	f.Lines = slices.Insert(f.Lines, f.depends.startLine+1, newLine)

	f.update()
}
