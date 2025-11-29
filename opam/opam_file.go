package opam

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	// Regex patterns for parsing opam files
	dependsRe       = regexp.MustCompile(`^\s*depends:\s*\[`)
	pinDependsRe    = regexp.MustCompile(`^\s*pin-depends:\s*\[`)
	closeBracketRe  = regexp.MustCompile(`^\s*\]`)
	beginIndirectRe = regexp.MustCompile(`^\s*##\s*begin indirect\s*$`)
	endIndirectRe   = regexp.MustCompile(`^\s*##\s*end\s*.*$`)
	// Matches: ["package.name" "git+https://...#commit"]
	pinDependLineRe = regexp.MustCompile(`^\s*\[\s*"([^"]+)"\s+"([^"]+)"\s*\]`)
)

type PinDepend struct {
	Package string // package name (e.g., rocq-iris)
	URL     string // URL (with git+https protocol)
	Commit  string // commit hash
}

type region struct {
	startLine int
	numLines  int
}

func (r region) empty() bool {
	return r.numLines <= 0
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
			f.depends.numLines = i - f.depends.startLine + 1
			inDepends = false
			continue
		}

		// Check for closing ] of pin-depends
		if inPinDepends && closeBracketRe.MatchString(line) {
			f.pinDepends.numLines = i - f.pinDepends.startLine + 1
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
				f.indirectPinDepends.numLines = i - indirectStart + 1
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
	// Strip .dev suffix from package name
	packageName = strings.TrimSuffix(packageName, ".dev")

	fullURL := matches[2]

	// Split URL into base and commit (split on #)
	url := fullURL
	commit := ""
	if idx := strings.IndexByte(fullURL, '#'); idx >= 0 {
		url = fullURL[:idx]
		commit = fullURL[idx+1:]
	}

	return &PinDepend{
		Package: packageName,
		URL:     url,
		Commit:  commit,
	}
}

// formatPinDependLine formats a PinDepend as an opam pin-depends line
func formatPinDependLine(dep PinDepend) string {
	// Normalize commit hash to first 15 characters
	commit := dep.Commit
	if len(commit) > 15 {
		commit = commit[:15]
	}

	fullURL := dep.URL
	if commit != "" {
		fullURL = dep.URL + "#" + commit
	}
	// Add .dev suffix to package name if not already present
	fullPackageName := dep.Package
	if !strings.HasSuffix(dep.Package, ".dev") {
		fullPackageName = dep.Package + ".dev"
	}
	// Use spacing similar to the example: package name padded with spaces between quotes
	// Total width is package name in quotes (package + 2 for quotes) padded to 27 chars
	quotedPkg := "\"" + fullPackageName + "\""
	padding := ""
	if len(quotedPkg) < 27 {
		padding = strings.Repeat(" ", 27-len(quotedPkg))
	}
	return fmt.Sprintf("  [%s%s \"%s\"]", quotedPkg, padding, fullURL)
}

func (f *OpamFile) ListPinDepends() []PinDepend {
	if f.pinDepends.empty() {
		return nil
	}

	var deps []PinDepend
	start := f.pinDepends.startLine + 1
	end := f.pinDepends.startLine + f.pinDepends.numLines - 1

	for i := start; i < end; i++ {
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

	newLine := formatPinDependLine(dep)

	start := f.pinDepends.startLine + 1
	end := f.pinDepends.startLine + f.pinDepends.numLines - 1

	// Search for existing entry and replace it
	foundIndex := -1
	for i := start; i < end; i++ {
		existingDep := parsePinDependLine(f.Lines[i])
		if existingDep != nil && existingDep.Package == dep.Package {
			foundIndex = i
			break
		}
	}

	// If found in indirect section, remove it from there and add to main section
	if foundIndex >= 0 && !f.indirectPinDepends.empty() &&
		foundIndex >= f.indirectPinDepends.startLine &&
		foundIndex < f.indirectPinDepends.startLine+f.indirectPinDepends.numLines {
		// Remove from indirect section
		newLines := make([]string, 0, len(f.Lines)-1)
		newLines = append(newLines, f.Lines[:foundIndex]...)
		newLines = append(newLines, f.Lines[foundIndex+1:]...)
		f.Lines = newLines

		err := f.findRegions()
		if err != nil {
			panic(err)
		}

		// Add to main section (after pin-depends: [ line)
		start = f.pinDepends.startLine + 1
		newLines = make([]string, 0, len(f.Lines)+1)
		newLines = append(newLines, f.Lines[:start]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, f.Lines[start:]...)
		f.Lines = newLines
	} else if foundIndex >= 0 {
		// Found in main section, just replace it
		f.Lines[foundIndex] = newLine
	} else {
		// Not found anywhere, add it after the pin-depends: [ line
		newLines := make([]string, 0, len(f.Lines)+1)
		newLines = append(newLines, f.Lines[:start]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, f.Lines[start:]...)
		f.Lines = newLines
	}

	err := f.findRegions()
	if err != nil {
		panic(err)
	}
}

func (f *OpamFile) GetIndirect() []PinDepend {
	if f.indirectPinDepends.empty() {
		return nil
	}

	var deps []PinDepend
	start := f.indirectPinDepends.startLine + 1                               // Skip "## begin indirect" line
	end := f.indirectPinDepends.startLine + f.indirectPinDepends.numLines - 1 // Skip "## end" line

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
		indirectStart := f.indirectPinDepends.startLine
		indirectEnd := f.indirectPinDepends.startLine + f.indirectPinDepends.numLines

		// Check if package exists in main pin-depends (outside indirect section)
		for i := start; i < f.pinDepends.startLine+f.pinDepends.numLines-1; i++ {
			// Skip lines in indirect section
			if !f.indirectPinDepends.empty() && i >= indirectStart && i < indirectEnd {
				continue
			}

			existingDep := parsePinDependLine(f.Lines[i])
			if existingDep != nil && existingDep.Package == indirect.Package {
				// Update the existing entry
				f.Lines[i] = formatPinDependLine(indirect)
				found = true
				break
			}
		}

		// Only add to indirect section if not found in main section
		if !found {
			filteredIndirects = append(filteredIndirects, indirect)
		}
	}

	var newLines []string

	// If there's already an indirect region, replace it
	if !f.indirectPinDepends.empty() {
		// Build new indirect section
		indirectLines := []string{"  ## begin indirect"}
		for _, dep := range filteredIndirects {
			indirectLines = append(indirectLines, formatPinDependLine(dep))
		}
		indirectLines = append(indirectLines, "  ## end")

		// Replace the indirect region
		start := f.indirectPinDepends.startLine
		end := f.indirectPinDepends.startLine + f.indirectPinDepends.numLines

		newLines = make([]string, 0, len(f.Lines)-f.indirectPinDepends.numLines+len(indirectLines))
		newLines = append(newLines, f.Lines[:start]...)
		newLines = append(newLines, indirectLines...)
		newLines = append(newLines, f.Lines[end:]...)
	} else {
		// Add new indirect section before the closing ] of pin-depends
		indirectLines := []string{
			"",
			"  ## begin indirect",
		}
		for _, dep := range filteredIndirects {
			indirectLines = append(indirectLines, formatPinDependLine(dep))
		}
		indirectLines = append(indirectLines, "  ## end")

		// Insert before the closing ] of pin-depends
		insertPos := f.pinDepends.startLine + f.pinDepends.numLines - 1

		newLines = make([]string, 0, len(f.Lines)+len(indirectLines))
		newLines = append(newLines, f.Lines[:insertPos]...)
		newLines = append(newLines, indirectLines...)
		newLines = append(newLines, f.Lines[insertPos:]...)
	}

	f.Lines = newLines
	err := f.findRegions()
	if err != nil {
		panic(err)
	}
}
