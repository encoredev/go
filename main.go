package main

import (
	"flag"
	"log"
	"os"

	"go.encore.dev/go/builder"
)

func main() {
	goos := flag.String("goos", "", "GOOS")
	goarch := flag.String("goarch", "", "GOARCH")
	dst := flag.String("dst", "dist", "build destination")
	upload := flag.Bool("upload", false, "If set, upload built artifacts to GCS")
	flag.Parse()

	if *goos == "" || *goarch == "" || *dst == "" {
		log.Fatalf("missing -dst %q, -goos %q, or -goarch %q", *dst, *goos, *goarch)
	}

	root, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if err := builder.BuildEncoreGo(*goos, *goarch, root, *dst, *upload); err != nil {
		log.Fatalln(err)
	}
}
