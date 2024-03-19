package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type Builder struct {
	GOOS        string
	GOARCH      string
	goroot      string
	gorootFinal string
	dst         string
	crossBuild  bool
}

func (b *Builder) PrepareWorkdir() error {
	if err := os.RemoveAll(b.dst); err != nil {
		return err
	} else if err := os.MkdirAll(b.dst, 0755); err != nil {
		return err
	}
	return nil
}

func (b *Builder) Build() error {
	var cmd *exec.Cmd
	switch b.GOOS {
	case "windows":
		cmd = exec.Command(".\\make.bat")
	default:
		cmd = exec.Command("bash", "./make.bash")
	}
	cmd.Dir = join(b.goroot, "src")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"GOROOT_FINAL=/encore",
		"GOARCH="+b.GOARCH,
		"GOOS="+b.GOOS,
	)

	// Enable cacheprog experiment on all platforms except Windows;
	// it's seemingly not supported there: we get weird build errors.
	if b.GOOS != "windows" {
		cmd.Env = append(cmd.Env, "GOEXPERIMENT=cacheprog")
	}

	return cmd.Run()
}

func (b *Builder) CopyOutput() error {
	key := b.GOOS + "_" + b.GOARCH
	filesToCopy := []string{
		join("pkg", "include"),
		join("pkg", "tool", key),
		"lib",
		"src",
		"LICENSE",
		"go.env",
	}

	// Cross-compilation puts binaries under bin/goos_goarch instead.
	if b.crossBuild {
		// Copy go binary from bin/goos_goarch to bin/
		src := join(b.goroot, "bin", key, "go")
		dst := join(b.dst, "bin", "go")
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if out, err := exec.Command("cp", src, dst).CombinedOutput(); err != nil {
			return fmt.Errorf("copy go: %s", out)
		}
	} else {
		filesToCopy = append(filesToCopy, join("bin", "go"+exe))
	}

	filesToCopy = append(filesToCopy, all(join("pkg", "tool", key),
		"addr2line"+exe, "asm"+exe, "buildid"+exe, "cgo"+exe, "compile"+exe,
		"link"+exe, "pack"+exe, "test2json"+exe, "vet"+exe, "cover"+exe,
	)...)

	for _, c := range filesToCopy {
		src := join(b.goroot, c)
		dst := join(b.dst, c)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("copy %s: %v", c, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if out, err := exec.Command("cp", "-r", src, dst).CombinedOutput(); err != nil {
			return fmt.Errorf("copy %s: %s", c, out)
		}
	}

	return nil
}

func (b *Builder) CleanOutput() error {
	key := b.GOOS + "_" + b.GOARCH
	rm := []string{
		join("pkg", key, "cmd"),
	}

	for _, r := range rm {
		dst := join(b.dst, r)
		if _, err := os.Stat(dst); err == nil {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("clean %s: %v", r, err)
			}
		}
	}

	return nil
}

func (b *Builder) Upload() error {
	ctx := context.Background()
	creds := os.Getenv("ENCORE_RELEASER_GCS_KEY")
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(creds)))
	if err != nil {
		return err
	}

	// Tar the artifacts.
	goVersion, err := readBuiltVersion()
	if err != nil {
		return fmt.Errorf("unable to read built version: %v", err)
	}

	// Create a tar.gz file.
	fmt.Println("Creating tar.gz file...")
	filename := fmt.Sprintf("%s-%s_%s.tar.gz", goVersion, b.GOOS, b.GOARCH)
	srcDir := filepath.Join(b.dst, b.GOOS+"_"+b.GOARCH)
	cmd := exec.Command("tar", "-czvf", filename, "-C", srcDir, ".")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar: %v", err)
	}

	objectPath := fmt.Sprintf("encore-go/%s/%s-%s.tar.gz", goVersion, b.GOOS, b.GOARCH)
	obj := client.Bucket("encore-releaser").Object(objectPath)

	{
		input, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("unable to open file: %v", err)
		}
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		w := obj.NewWriter(ctx)
		if _, err := io.Copy(w, input); err != nil {
			return fmt.Errorf("unable to copy file: %v", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("unable to complete upload: %v", err)
		}
	}

	return nil
}

func join(strs ...string) string {
	return filepath.Join(strs...)
}

func all(src string, all ...string) []string {
	var res []string
	for _, a := range all {
		res = append(res, join(src, a))
	}
	return res
}

func BuildEncoreGo(goos, goarch, root, dst string, upload bool) error {
	if _, err := os.Stat(filepath.Join(root, "go", "src", "make.bash")); err != nil {
		return fmt.Errorf("unexpected location for build script, expected in encore-go root")
	}

	if err := os.Chdir(root); err != nil {
		return err
	}

	dst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}

	if goos == "windows" {
		exe = ".exe"
	}

	b := &Builder{
		GOOS:       goos,
		GOARCH:     goarch,
		goroot:     join(root, "go"),
		dst:        join(dst, goos+"_"+goarch, "encore-go"),
		crossBuild: runtime.GOOS != goos || runtime.GOARCH != goarch,
	}

	for _, f := range []func() error{
		b.PrepareWorkdir,
		b.Build,
		b.CopyOutput,
		b.CleanOutput,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	if upload {
		if err := b.Upload(); err != nil {
			return err
		}
	}

	return nil
}

// exe suffix
var exe string

func readBuiltVersion() (version string, err error) {
	var str string
	if isfile("go/VERSION") {
		// If we're building from a release branch, we use this as the base
		str, err = readfile("go/VERSION")
		if err != nil {
			return "", fmt.Errorf("unable to read file: %w", err)
		}
		// Then we repeat the replace we do within the src/cmd/dist/build.go
		str = strings.Replace(str, "go1.", "encore-go1.", 1)
	} else {
		// Otherwise we read the cache file which would be created by the build process
		// if there was no VERSION file present
		str, err = readfile("go/VERSION.cache")
		if err != nil {
			return "", fmt.Errorf("unable to read file: %w", err)
		}
	}

	// With our patches there must always be an `encore-go1.xx` version in this string
	// (there may be other bits, like "devel" or "beta" which we don't care about)
	re, err := regexp.Compile("(encore-go[^ ]+)")
	if err != nil {
		return "", fmt.Errorf("unable to compile regex: %w", err)
	}
	version = re.FindString(str)
	if version == "" {
		return "", fmt.Errorf("unable to extract version, read: %s", version)
	}

	// In Go 1.21 the time was added as the second line of the VERSION file
	// so we only want the first line
	version, _, _ = strings.Cut(version, "\n")
	version = strings.TrimSpace(version)

	return version, nil
}

// isfile reports whether p names an existing file.
func isfile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// readfile returns the content of the named file.
func readfile(file string) (string, error) {
	data, err := os.ReadFile(file)
	return strings.TrimRight(string(data), " \t\r\n"), err
}
