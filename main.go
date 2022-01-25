package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"go.encore.dev/go/builder"
)

func main() {
	goos := flag.String("goos", "", "GOOS")
	goarch := flag.String("goarch", "", "GOARCH")
	dst := flag.String("dst", "dist", "build destination")
	readVersion := flag.Bool("read-built-version", false, "If set we'll simply parse go/VERSION.cache and return the Go verison")
	flag.Parse()

	if *readVersion {
		readBuiltVersion()

		return
	}

	if *goos == "" || *goarch == "" || *dst == "" {
		log.Fatalf("missing -dst %q, -goos %q, or -goarch %q", *dst, *goos, *goarch)
	}

	root, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if err := builder.BuildEncoreGo(*goos, *goarch, root, *dst); err != nil {
		log.Fatalln(err)
	}
}

func readBuiltVersion() {
	bytes, err := ioutil.ReadFile("go/VERSION.cache")
	if err != nil {
		log.Fatalf("Unable to read built version: %+v", err)
	}

	str := string(bytes)

	re, err := regexp.Compile("(encore-go[^ ]+)")
	if err != nil {
		log.Fatalf("Unable to compile regex: %+v", err)
	}

	version := re.FindString(str)
	if version == "" {
		log.Fatalf("Unable to extract version from `go/VERSION.cache` read: %s", str)
	}

	fmt.Println(version)
}
