package test

import (
	"bufio"
	"bytes"
	"cmd/go/internal/base"
	"cmd/go/internal/fsys"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// copyCoverageProfile copies the coverage profile report into
// the output file while restoring the original filenames from
// any overlays that where applied by Encore.
//
// Normally the coverage profile is copied using in [mergeCoverProfile] using:
//
//	_, err = io.Copy(coverMerge.f, r)
//
// However, this doesn't work with overlays because the coverage
// profile contains the overlay file names, so we replace that line with::
//
//	err = copyCoverageProfile(r, coverMerge.f)
//
// It restores the original file names from the overlay file.
func copyCoverageProfile(from io.Reader, to io.Writer) (err error) {
	encoreOnce.Do(func() { err = initEncoreReverseMap() })
	if err != nil {
		println("encore: error initializing overlay reverse map: " + err.Error())
		return err
	}

	b, err := io.ReadAll(from)
	if err != nil {
		return err
	}

	// Read the coverage profile line by line.
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text() + "\n"

		// Update the line if we have the filename is an overlay file
		// pointing back to the original file.
		filename, rest, found := strings.Cut(line, ":")
		if found {
			if mappedFilename, ok := encoreOverlayReverseMap[filename]; ok {
				line = mappedFilename + ":" + rest
			}
		}

		// Write the line to the output file.
		_, err := to.Write([]byte(line))
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	encoreOverlayReverseMap map[string]string
	encoreOnce              sync.Once
)

// initEncoreReverseMap initializes the encoreOverlayReverseMap
// which is a mapping of "[pkg]/[overlay_file].go" -> "[pkg]/[original_file].go"
//
// It does this by:
//  1. Reading the overlay file (fsys.OverlayFile)
//  2. Working out the packages that we're overlaying based on their file paths
//  3. Building a map of "[pkg]/[overlay_file].go" -> "[pkg]/[original_file].go"
func initEncoreReverseMap() error {
	// 1) First read the overlay file
	var overlayJSON fsys.OverlayJSON
	{ // This block is mostly copied from: cmd/go/internal/fsys/fsys.go Init

		// Read the overlay file
		b, err := os.ReadFile(fsys.OverlayFile)
		if err != nil {
			return fmt.Errorf("reading overlay file: %v", err)
		}

		// Parse the overlay file
		if err := json.Unmarshal(b, &overlayJSON); err != nil {
			return fmt.Errorf("parsing overlay JSON: %v", err)
		}
	}

	// 2) Work out the packages that we're overlaying

	// Pkg describes a single package, compatible with the JSON output from 'go list'; see 'go help list'.
	type Pkg struct {
		ImportPath string
		Dir        string
		Error      *struct {
			Err string
		}
	}
	pkgs := make(map[string]*Pkg)

	{ // This block is mostly copied from: cmd/cover/func.go findPkgs
		pkgList := make([]string, 0, len(overlayJSON.Replace))
		pkgSet := make(map[string]struct{})
		for baseFile := range overlayJSON.Replace {
			// We only care about the baseFile, as overlays are reported
			// to be in the original files package
			baseDir := filepath.Dir(canonicalize(baseFile))
			if _, ok := pkgSet[baseDir]; !ok {
				pkgSet[baseDir] = struct{}{}
				pkgList = append(pkgList, baseDir)
			}
		}

		// Now run go list to find the location of every package we care about.
		goTool := filepath.Join(runtime.GOROOT(), "bin/go")
		cmd := exec.Command(goTool, append([]string{"list", "-e", "-json"}, pkgList...)...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("cannot run go list: %v\n%s", err, stderr.Bytes())
		}
		dec := json.NewDecoder(bytes.NewReader(stdout))
		for {
			var pkg Pkg
			err := dec.Decode(&pkg)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("decoding go list json: %v", err)
			}

			if pkg.Error == nil && pkg.ImportPath != "" && pkg.Dir != "" {
				pkgs[pkg.Dir] = &pkg
			}
		}
	}

	// 3) Now build the reverse map
	encoreOverlayReverseMap = make(map[string]string)
	for basePath, overlayPath := range overlayJSON.Replace {
		// Find the package for the original file
		basePath = canonicalize(basePath)
		pkg, found := pkgs[filepath.Dir(basePath)]
		if !found {
			// Some of Encore's internals are not in the go list, so we ignore them
			continue
		}

		// If the original file and overlay file are reported with different filenames
		// then lets add a mapping to the reverse map.
		reportedFile := filepath.Join(pkg.ImportPath, filepath.Base(canonicalize(overlayPath)))
		originalFile := filepath.Join(pkg.ImportPath, filepath.Base(basePath))
		if reportedFile != originalFile {
			encoreOverlayReverseMap[reportedFile] = originalFile
		}
	}

	return nil
}

// copied from cmd/go/internal/fsys/fsys.go
func canonicalize(path string) string {
	cwd := base.Cwd()

	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	if v := filepath.VolumeName(cwd); v != "" && path[0] == filepath.Separator {
		// On Windows filepath.Join(cwd, path) doesn't always work. In general
		// filepath.Abs needs to make a syscall on Windows. Elsewhere in cmd/go
		// use filepath.Join(cwd, path), but cmd/go specifically supports Windows
		// paths that start with "\" which implies the path is relative to the
		// volume of the working directory. See golang.org/issue/8130.
		return filepath.Join(v, path)
	}

	// Make the path absolute.
	return filepath.Join(cwd, path)
}
