package hashio

import (
	"crypto/md5" //nolint
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
)

const size = 512

type HashFunc func([]byte) ([]byte, error)

var ErrHashFuncNotFound = errors.New("hash func not found")

// ReadAll reads in blocks by buf size and hashes
func ReadAll(r io.Reader, hasher hash.Hash) ([]byte, error) {
	buf := make([]byte, size)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				if n > 0 {
					hasher.Write(buf[:n])
				}
				break
			}

			return nil, fmt.Errorf("read: %w", err)
		}

		hasher.Write(buf[:n])
	}

	return hasher.Sum(nil), nil
}

// ReadFile takes the virtual file system interface fs.FS and fully reads the contents of the file,
// then applies a HashFunc to it
func ReadFile(fsys fs.FS, fileName string, hashFunc HashFunc) ([]byte, error) {
	if hashFunc == nil {
		return nil, ErrHashFuncNotFound
	}

	input, err := fs.ReadFile(fsys, fileName)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", fileName, err)
	}

	output, err := hashFunc(input)
	if err != nil {
		return nil, fmt.Errorf("call HashFunc: %w", err)
	}

	return output, nil
}

func HashSumFunc(hasher func() hash.Hash) HashFunc {
	return func(in []byte) ([]byte, error) {
		h := hasher()
		if _, err := h.Write(in); err != nil {
			return nil, fmt.Errorf("%T(hashfile.Hash) write: %w", h, err)
		}

		return h.Sum(nil), nil
	}
}

func MD5() func() hash.Hash {
	return func() hash.Hash {
		return md5.New()
	}
}

func SHA1() func() hash.Hash {
	return func() hash.Hash {
		return sha1.New()
	}
}

func MD5HashFunc() HashFunc {
	return HashSumFunc(func() hash.Hash {
		return md5.New()
	})
}

func SHA1HashFunc() HashFunc {
	return HashSumFunc(func() hash.Hash {
		return sha1.New()
	})
}
