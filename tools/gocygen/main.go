package main

import (
	"context"
	"errors"
	"flag"
	"hash"
	"log"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/robotomize/gokuu/internal/gen"
	"github.com/robotomize/gokuu/internal/hashio"
	"github.com/robotomize/gokuu/internal/logging"
)

var flagGen = flag.NewFlagSet("flaggen", flag.ContinueOnError)

var (
	path     = flagGen.String("target", "", "path to the folder with the generated files")
	hashFunc = flagGen.String("hash", "", "hash alg for compare files, variants: md5, sha1")
)

var defaultHasherFunc = hashio.MD5()

func main() {
	ctx := logging.WithLogger(context.Background(), logging.NewLogger("Gocygen: ", log.Lmsgprefix))
	logger := logging.FromContext(ctx)

	if err := flagGen.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("flag parse: %v", err)
	}

	if *path == "" {
		logger.Fatal("use -target <path> - path to the folder with the generated files")
	}

	var hasherFunc func() hash.Hash
	switch *hashFunc {
	case "md5":
		hasherFunc = hashio.MD5()
	case "sha1":
		hasherFunc = hashio.SHA1()
	default:
		hasherFunc = defaultHasherFunc
	}

	if err := gen.Generate(*path, hasherFunc); err != nil {
		var multiErr *multierror.Error
		if errors.As(err, &multiErr) {
			for _, wrErr := range multiErr.WrappedErrors() {
				if !errors.Is(wrErr, gen.ErrHashingContentEqual) {
					logger.Fatal(multiErr)
				}

				logger.Printf("warning: %v", gen.ErrHashingContentEqual)
			}
			goto WarningErr
		}

		logger.Fatal(err)
	}

WarningErr:
	logger.Printf("files were completed successfully, generated files are placed in %s", *path)
}
