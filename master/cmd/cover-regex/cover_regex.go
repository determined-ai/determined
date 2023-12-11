//nolint:all
package main

// Heavily inspired from https://github.com/golang/go/blob/aae7734658e5f302c0e3a10f6c5c596fd384dbd7/src/cmd/cover/html.go

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/tools/cover"
)

var regexesAndMessages = [][]string{
	{`.*Scan\((ctx|context)`, "Bun 'Scan()' queries must be covered by an integration test"},
	{`.*Exec\((ctx|context)`, "Bun 'Exec()' queries must be covered by an integration test"},
	{`.*Count\((ctx|context)`, "Bun 'Count()' queries must be covered by an integration test"},
	{`.*Exists\((ctx|context)`, "Bun 'Exists()' queries must be covered by an integration test"},
	{`.*ScanAndCount\((ctx|context)`, "Bun 'ScanAndCount()' queries must be covered by an integration test"},
}

type regexWithInfo struct {
	regex           *regexp.Regexp
	regexString     string
	humanReadable   string
	coveredCount    int
	unconveredCount int
}

func main() {
	if len(os.Args) != 2 {
		panic(fmt.Sprintf("Needs two file args got %v instead", os.Args))
	}

	if err := doLinter(os.Args[1]); err != nil {
		panic(err)
	}
}

func doLinter(profile string) error {
	profiles, err := cover.ParseProfiles(profile)
	if err != nil {
		return err
	}

	var regexWithInfos []*regexWithInfo
	for _, r := range regexesAndMessages {
		regexWithInfos = append(regexWithInfos, &regexWithInfo{
			regex:         regexp.MustCompile(r[0]),
			regexString:   r[0],
			humanReadable: r[1],
		})
	}

	dirs, err := findPkgs(profiles)
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		fn := profile.FileName

		file, err := findFile(dirs, fn)
		if err != nil {
			return err
		}
		src, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("can't read %q: %v", fn, err)
		}

		bounds := profile.Boundaries(src)

		regexGen(file, src, bounds, regexWithInfos)
	}

	fmt.Println("=====================================")
	fmt.Println("Summaries")
	fmt.Print("=====================================\n\n")

	shouldFail := false
	for _, r := range regexWithInfos {
		passOrFail := "PASS"
		if r.unconveredCount > 0 {
			passOrFail = "FAIL"
			shouldFail = true
		}

		fmt.Printf(`%s Regex:"%s" %s CoveredLines (%d) / Total Lines (%d) Percent "%.2f%%`+"\n",
			passOrFail, r.regexString, r.humanReadable, r.coveredCount,
			r.coveredCount+r.unconveredCount,
			100.0*float64(r.coveredCount)/float64(r.coveredCount+r.unconveredCount))
	}

	fmt.Println("\n=====================================")
	fmt.Println("Result")
	fmt.Println("=====================================")

	if shouldFail {
		fmt.Println("FAILED")
		os.Exit(1)
	} else {
		fmt.Println("PASSED")
		os.Exit(0)
	}

	return nil
}

func regexGen(
	fileName string,
	src []byte,
	boundaries []cover.Boundary,
	regexWithInfos []*regexWithInfo,
) {
	lineIndex := 0
	covered := false

	boundText := ""
	for i := range src {
		for len(boundaries) > 0 && boundaries[0].Offset == i {
			b := boundaries[0]

			if b.Start {
				for _, r := range regexWithInfos {
					if r.regex.MatchString(boundText) {
						if covered {
							r.coveredCount++
						} else {
							r.unconveredCount++
							fmt.Printf("%s %s:%d\n%s\n\n",
								r.humanReadable, fileName, lineIndex,
								strings.ReplaceAll(">>>"+boundText, "\n", "\n>>>"))
						}
					}
				}

				// Needs to go after since the bounds applies to next.
				covered = b.Count > 0
				boundText = ""
			}

			boundaries = boundaries[1:]
		}

		boundText += string(src[i])
		if src[i] == '\n' {
			lineIndex++
		}
	}
}

// Pkg describes a single package, compatible with the JSON output from 'go list'; see 'go help list'.
type Pkg struct {
	ImportPath string
	Dir        string
	Error      *struct {
		Err string
	}
}

func findPkgs(profiles []*cover.Profile) (map[string]*Pkg, error) {
	// Run go list to find the location of every package we care about.
	pkgs := make(map[string]*Pkg)
	var list []string
	for _, profile := range profiles {
		if strings.HasPrefix(profile.FileName, ".") || filepath.IsAbs(profile.FileName) {
			// Relative or absolute path.
			continue
		}
		pkg := path.Dir(profile.FileName)
		if _, ok := pkgs[pkg]; !ok {
			pkgs[pkg] = nil
			list = append(list, pkg)
		}
	}

	if len(list) == 0 {
		return pkgs, nil
	}

	// Note: usually run as "go tool cover" in which case $GOROOT is set,
	// in which case runtime.GOROOT() does exactly what we want.
	goTool := filepath.Join(runtime.GOROOT(), "bin/go")
	cmd := exec.Command(goTool, append([]string{"list", "-e", "-json"}, list...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot run go list: %v\n%s", err, stderr.Bytes())
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))
	for {
		var pkg Pkg
		err := dec.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding go list json: %v", err)
		}
		pkgs[pkg.ImportPath] = &pkg
	}
	return pkgs, nil
}

// findFile finds the location of the named file in GOROOT, GOPATH etc.
func findFile(pkgs map[string]*Pkg, file string) (string, error) {
	if strings.HasPrefix(file, ".") || filepath.IsAbs(file) {
		// Relative or absolute path.
		return file, nil
	}
	pkg := pkgs[path.Dir(file)]
	if pkg != nil {
		if pkg.Dir != "" {
			return filepath.Join(pkg.Dir, path.Base(file)), nil
		}
		if pkg.Error != nil {
			return "", errors.New(pkg.Error.Err)
		}
	}
	return "", fmt.Errorf("did not find package for %s in go list output", file)
}
