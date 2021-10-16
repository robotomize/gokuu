package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/robotomize/gokuu/internal/hashio"
	"github.com/robotomize/gokuu/internal/logging"
)

const (
	defaultUserAgent      = "go-currency/0.0.0"
	defaultRequestTimeout = 10 * time.Second
)

const (
	cldrEnCurrencyURL = "https://raw.githubusercontent.com/unicode-org/cldr-json/master/cldr-json/cldr-numbers-modern/main/en-001/currencies.json"
	isoCurrencyURL    = "https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list_one.xml"
)

const (
	currencyCodesFile = "currency_codes.xml"
	currencyNamesFile = "currency_names.json"
)

var ErrHashingContentEqual = errors.New("hash of the fetching file is equivalent to the previous version")
var (
	defaultHasherFunc = hashio.MD5()
	flagUpd           = flag.NewFlagSet("flagupd", flag.ContinueOnError)
)

var (
	path     = flagUpd.String("target", "", "path to the folder with the assets")
	hashFunc = flagUpd.String("hash", "", "hash alg for compare files, variants: md5, sha1")
)

func main() {
	ctx := logging.WithLogger(context.Background(), logging.NewLogger("Gocyupd: ", log.Lmsgprefix))
	logger := logging.FromContext(ctx)

	if err := flagUpd.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("flag parse: %v", err)
	}

	if *path == "" {
		logger.Fatal("use -target <path> path to the folder with the assets")
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

	if err := realMain(ctx, *path, hasherFunc); err != nil {
		var multiErr *multierror.Error
		if errors.As(err, &multiErr) {
			for _, wrErr := range multiErr.WrappedErrors() {
				if !errors.Is(wrErr, ErrHashingContentEqual) {
					logger.Fatal(multiErr)
				}

				logger.Printf("warning: %v", ErrHashingContentEqual)
			}
			return
		}

		logger.Fatal(err)
	}
}

func realMain(ctx context.Context, path string, hasherFunc func() hash.Hash) error {
	var multiErr multierror.Group
	client := &http.Client{Transport: &http.Transport{
		MaxIdleConns:          20000,
		MaxIdleConnsPerHost:   1000,
		DisableCompression:    true,
		IdleConnTimeout:       5 * time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}}

	multiErr.Go(func() error {
		u, err := url.Parse(isoCurrencyURL)
		if err != nil {
			return fmt.Errorf("error: url parse: %w", err)
		}

		client := NewHTTPClient(u, client)

		path := filepath.Join(path, currencyCodesFile)

		if err := sync(ctx, client, path, hasherFunc); err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		return nil
	})

	multiErr.Go(func() error {
		u, err := url.Parse(cldrEnCurrencyURL)
		if err != nil {
			return fmt.Errorf("error: url parse: %w", err)
		}

		client := NewHTTPClient(u, client)

		path := filepath.Join(path, currencyNamesFile)

		if err := sync(ctx, client, path, hasherFunc); err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		return nil
	})

	if err := multiErr.Wait(); err != nil {
		return fmt.Errorf("syncing error: %w", err)
	}

	return nil
}

func sync(ctx context.Context, client *HTTPClient, fileName string, hasherFunc func() hash.Hash) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	var (
		oldHash, newHash []byte
		mode             os.FileMode = 0600
	)

	body, err := client.do(ctx)
	if err != nil {
		return fmt.Errorf("http client do: %w", err)
	}

	oldHash, err = hashingFile(fileName, hasherFunc)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("hashing file content: %w", err)
		}
	}

	newHash, err = hashio.ReadAll(bytes.NewReader(body), hasherFunc())
	if err != nil {
		return fmt.Errorf("hashing body content: %w", err)
	}

	if bytes.Equal(newHash, oldHash) {
		return ErrHashingContentEqual
	}

	info, err := os.Stat(fileName)
	if err != nil {
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) {
			return fmt.Errorf("hashing file content: %w", err)
		}
	}

	if info != nil {
		mode = info.Mode()
	}

	if err := os.WriteFile(fileName, body, mode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func NewHTTPClient(u *url.URL, client *http.Client) *HTTPClient {
	return &HTTPClient{
		url:    u,
		client: client,
	}
}

type HTTPClient struct {
	url    *url.URL
	client *http.Client
}

func (h *HTTPClient) do(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d %s", resp.StatusCode, resp.Status)
	}

	var reader io.ReadCloser

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable create gzip.NewReader: %w", err)
		}

		reader = gz
		defer reader.Close()
	default:
		reader = resp.Body
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read http body: %w", err)
	}

	return b, nil
}

func hashingFile(fileName string, hasherFunc func() hash.Hash) ([]byte, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, nil
	}
	hasher := hasherFunc()
	b, err := hashio.ReadAll(file, hasher)
	if err != nil {
		return nil, fmt.Errorf("hashing file content: %w", err)
	}

	return b, nil
}
