<div align="center">
  <a href="https://encore.dev" alt="encore"><img width="189px" src="https://encore.dev/assets/img/logo.svg"></a>
  <h3><a href="https://encore.dev">Encore – The Backend Development Engine</a></h3>
</div>

# Encore Rolling Go Fork

This is [Encore's](https://encore.dev) rolling fork of Go with added automatic instrumentation for local development.

This branch contains the raw patches, which allow us to re-apply them ontop of new Go releases.

This system produces [reproducible builds](https://reproducible-builds.org/) of the patched Go runtime. This means you can clone this repository, run the commands listed below and reproduce an identical go binary that we ship with our tooling.

## How to Use

1. Checkout this repository and then initial the Git submodule; `git submodule init && git submodule update`
2. Create a fresh patched version of the Go runtime, passing in the Go version you want;  `./apply_patch.bash 1.17`
   (Note: instead of a Go version number, you can pass in `master` to build against the latest Go development commit)
3. Run our build script using; `go run . --goos "darwin" --goarch "arm64"`
   (replacing the OS and arch parameters to match your requirement)
4. Verify your go was build; `./dist/darwin_arm64/encore-go/bin/go version`
    The output you see should look like this, looking for the `encore-go` string;
   `go version encore-go1.17.6 encore-go1.17-4d15582aff Thu Jan 6 19:06:43 2022 +0000 darwin/arm64`

## Directory Structure

This branch is broken up into three main folders;
- `overlay`; this folder contains brand-new Encore specific files which should be copied into the `src` path of a fresh Go Release
- `patches`; this folder contains patch files to modify the existing Go source code
- `go`; a submodule checkout of https://go.googlesource.com/go which we apply the above two folders against 
