package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		"GOOS="+b.GOOS)
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

func BuildEncoreGo(goos, goarch, root, dst string) error {
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

	return nil
}

// exe suffix
var exe string
