package hashio

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type fakeReader struct{}

func (f *fakeReader) Read(_ []byte) (n int, err error) { return 0, errors.New("io error") }

func TestReadAll(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		err         error
		r           io.Reader
		hasherFunc  func() hash.Hash
		expectedStr string
		sourceStr   string
	}{
		{
			name:        "test_read_all_md5",
			hasherFunc:  MD5(),
			sourceStr:   "hello world",
			expectedStr: "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:        "test_read_all_sha1",
			hasherFunc:  SHA1(),
			sourceStr:   "hello world",
			expectedStr: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:        "test_read_all_sha1",
			hasherFunc:  SHA1(),
			sourceStr:   "hello world",
			expectedStr: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
			r:           &fakeReader{},
			err:         errors.New("io error"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				var r io.Reader
				if tc.r != nil {
					r = tc.r
				} else {
					r = bytes.NewReader([]byte(tc.sourceStr))
				}
				h := tc.hasherFunc()
				got, err := ReadAll(r, h)
				if tc.err != nil && err == nil {
					t.Fatalf("read all: %v", err)
				}

				if err != nil {
					if !errors.Is(err, tc.err) {
						if strings.Contains(err.Error(), tc.err.Error()) {
							return
						}
						diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
						t.Errorf("mismatch (-want, +got):\n%s", diff)
					}
					return
				}

				if strings.Compare(tc.expectedStr, fmt.Sprintf("%x", got)) != 0 {
					diff := cmp.Diff(tc.expectedStr, fmt.Sprintf("%x", got))
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}
			},
		)
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		err         error
		fileName    string
		hashFunc    HashFunc
		expectedStr string
		sourceStr   string
	}{
		{
			name:        "test_read_file_md5",
			fileName:    "stat",
			hashFunc:    MD5HashFunc(),
			sourceStr:   "hello world",
			expectedStr: "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:        "test_read_file_sha1",
			fileName:    "stat",
			hashFunc:    SHA1HashFunc(),
			sourceStr:   "hello world",
			expectedStr: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:        "test_read_file_md5_with_helper",
			fileName:    "stat",
			hashFunc:    HashSumFunc(MD5()),
			sourceStr:   "hello world",
			expectedStr: "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:     "test_read_file_sha1_with_helper",
			fileName: "stat",
			hashFunc: HashSumFunc(
				func() hash.Hash {
					return sha1.New()
				},
			),
			sourceStr:   "hello world",
			expectedStr: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				fs := fstest.MapFS{
					tc.fileName: &fstest.MapFile{
						Data:    []byte(tc.sourceStr),
						Mode:    0o600,
						ModTime: time.Time{},
					},
				}

				got, err := ReadFile(fs, tc.fileName, tc.hashFunc)
				if tc.err != nil && err == nil {
					t.Fatalf("read all: %v", err)
				}

				if err != nil {
					if !errors.Is(err, tc.err) {
						if strings.Contains(err.Error(), tc.err.Error()) {
							return
						}
						diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
						t.Errorf("mismatch (-want, +got):\n%s", diff)
					}
					return
				}

				if strings.Compare(tc.expectedStr, fmt.Sprintf("%x", got)) != 0 {
					diff := cmp.Diff(tc.expectedStr, fmt.Sprintf("%x", got))
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}
			},
		)
	}
}
