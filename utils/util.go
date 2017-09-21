package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// NewUUID generates a random UUID according to RFC 4122
func NewUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// Variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80

	// Version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

//Unpack ...
func Unpack(source io.Reader, target string) error {
	archive, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()

	tarReader := tar.NewReader(archive)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		log.Println(path)
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// RetryWithBackoff takes a Backoff and a function to call that returns an error
// If the error is nil then the function will no longer be called.  If the error
// is Retriable then that will be used to determine if it should be retried
func RetryWithBackoff(backoff Backoff, fn func() error) error {
	var err error
	for err = fn(); true; err = fn() {
		retryable, isRetryable := err.(Retryable)

		if err == nil || isRetryable && !retryable.Retry() {
			return err
		}

		time.Sleep(backoff.Duration())
	}
	return err
}

// RetryNWithBackoff takes a Backoff, a maximum number of tries 'n', and a
// function that returns an error. The function is called until either it does
// not return an error or the maximum tries have been reached.
func RetryNWithBackoff(backoff Backoff, n int, fn func() error) error {
	var err error
	RetryWithBackoff(backoff, func() error {
		err = fn()
		n--
		if n == 0 {
			// Break out after n tries
			return nil
		}
		return err
	})
	return err
}
