package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/aes"
	"crypto/cipher"
	"encoding/pem"
	"bytes"
	"io/ioutil"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
	"strconv"
	"strings"
	"os/exec"
	
	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
)

// Constants
var(
	// extension for encrypted files
	EncryptedExtension = ".enc"
	GzipExtension = ".gz"
	LockKeyExtension = ".lockkey"
	DecryptedLockKeyLength = 65
	IvLength = 16
	AesLength = 32
	PadLength = 1
	DecryptTPMCommandFmt = "openssl rsautl -decrypt -keyform engine -engine tpm2tss -inkey %s"
	DecryptOpensslCommandFmt = "openssl rsautl -decrypt -inkey %s"
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

// return lock key name if available or empty string otherwise
func GetLockKeyName(cfg config.Config) (string, error) {
	if len(cfg.KeyName) == 0 {
		return "", nil
	}
	nameBytes, err := ioutil.ReadFile(cfg.KeyName)
	if err != nil {
		fmt.Printf("Could not key name from file (%s): \n%v", cfg.KeyName, err)
	}
	return string(nameBytes), nil
}

// return decryption key if available
func GetDecryptionKey(cfg config.Config) (*rsa.PrivateKey, string, error) {
	if len(cfg.KeyLocation) == 0 {
		return nil, "", errors.New("Cannot get decryption key, no key configured")
	}	
	keyInfo, err := ioutil.ReadFile(cfg.KeyLocation)
	// unpack key, binary format: [machine's lockkey name][null terminator][rsa key]
	if err != nil {
		return nil, "", err
	}
	nameLen := -1
	for i := 0; i < len(keyInfo); i ++ {
		//find null terminator for string
		if keyInfo[i] == 0 {
			nameLen = i
			break
		}
	}
	//separate lockkey name (expected RSA encrypted symmetric key info for this machine)
	lockKeyName := string(keyInfo[0 : nameLen]) + LockKeyExtension
	//separate RSA private key info for this machine and parse it
	pemString := keyInfo[nameLen + 1 : len(keyInfo)]
	block, _ := pem.Decode([]byte(pemString))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	return key, lockKeyName, nil
}

func isEncryptedPackage(archive *gzip.Reader) (bool, error) {
	hasEncrypted := false
	hasUnencrypted := false
	tarReader := tar.NewReader(archive)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return false, err
		}
		info := header.FileInfo()
		if !info.IsDir() {
			if filepath.Ext(header.Name) == EncryptedExtension {
				hasEncrypted = true
			} else if filepath.Ext(header.Name) == GzipExtension {
				hasUnencrypted = true
			}
		}
	}
	if hasEncrypted && hasUnencrypted {
		return false, errors.New("Package contains both encrypted and unencrypted payloads")
	} else if !hasEncrypted && !hasUnencrypted {
		return false, errors.New("Package contains neither encrypted nor unencrypted payloads")
	}
	return hasEncrypted, nil
}

func HasTPM2() (bool, error) {
	result, err := exec.Command("sh", "-c", "systemctl is-active tpm2-abrmd | grep -o inactive | wc -l").Output()
	if err != nil {
		return false, err
	}
	lineCount, err := strconv.Atoi(strings.Trim(string(result), "\n"))
	if err != nil {
		return false, err
	}
	hasTPM := lineCount == 0
	return hasTPM, nil
}

//Unpack package tarball
// Expected Manifest:
// <application_name>.tar.gz            (top level tarball)
//   - MANIFEST.JSON                    (LTC parsable package info)
//   - <lockfile_name 0..n>.lockfile    (RSA encrypted symmetric key files, there are many... 1 per machine)
//   - <application_name>.tar.gz<.enc>  (application payload - .enc indicates encrypted by symmetric key in .lockfile)
func Unpack(source io.Reader, target string, cfg config.Config) error {
	var unencryptedReader io.Reader
	var lockkeyData, encryptedData []byte
	//open top level of tarball
	fmt.Println("Unpacking application pacakge...")
	topArchive, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer topArchive.Close()
	tarReader := tar.NewReader(topArchive)
//	key, lockKeyName, getKeyErr := GetDecryptionKey(cfg)
//	fmt.Println("  Found decryption key")
	lockKeyName, getKeyErr := GetLockKeyName(cfg)
	fmt.Println("  Package contents:")
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		info := header.FileInfo()
		fmt.Printf("    %s\n", header.Name)
		if !info.IsDir() {
			if filepath.Ext(header.Name) == EncryptedExtension {
				if encryptedData != nil {
					return errors.New("Application package malformed: multiple encrypted payloads")
				}
				encryptedData, err = ioutil.ReadAll(tarReader)
				if err != nil {
					return err
				}
			} else if filepath.Ext(header.Name) == GzipExtension {
				if unencryptedReader != nil {
					return errors.New("Application package malformed: multiple unencrypted payloads.  This may be a deprecated package format.")
				}
				data, err := ioutil.ReadAll(tarReader)
				if err != nil {
					return err
				}
				unencryptedReader = bytes.NewReader(data)
			} else if lockKeyName != "" && filepath.Base(header.Name) == lockKeyName {
				if lockkeyData != nil {
					return errors.New("Application package malformed: multiple machine lockkeys")
				}
				lockkeyData, err = ioutil.ReadAll(tarReader)
				if err != nil {
					return err
				}
			}
		}
	}
	
	if encryptedData == nil && unencryptedReader == nil {
		return errors.New("Application package malformed: no package payload found")
	} else if encryptedData != nil && unencryptedReader != nil {
		return errors.New("Application package malformed: contains encrypted and clear payloads")
	} else if encryptedData != nil && getKeyErr != nil {
		return getKeyErr //TODO: maybe wrap this err message to provide more context
	} else if encryptedData != nil && lockkeyData == nil {
			errString := fmt.Sprintf("Application package malformed: encrypted package, but no lockkey for this machine (looking for %s)",
				lockKeyName)
		return errors.New(errString)
	} else if encryptedData != nil && lockkeyData != nil {
		fmt.Println("  This is an encrypted package, decrypting...")
		// lockkey & parse padding, key, and iv
		//aesPadKeyIv, err := rsa.DecryptPKCS1v15(nil, key, lockkeyData)
		unlockKeyCommand := fmt.Sprintf(DecryptOpensslCommandFmt, cfg.KeyLocation)
		hasTPM, err := HasTPM2()
		if err != nil {
			fmt.Printf("    Error determining if this device uses TPM2.0: %s\n", err.Error())
			return err
		}
		if hasTPM {
			fmt.Println("  TPM2.0 Detected, decrypting using TPM locked private key")
			unlockKeyCommand = fmt.Sprintf(DecryptTPMCommandFmt, cfg.KeyLocation)
		} else {
			fmt.Println("  NO TPM FOUND, decrypting using openssl generated private key")
		}
		cmd := exec.Command("sh", "-c", unlockKeyCommand)
		// outPipe, err := cmd.StdoutPipe()
		// if err != nil {
		// 	fmt.Printf("    Error decrypting lock key (get stdout): %s\n", err.Error)
		// 	return err
		// }
		inPipe, err := cmd.StdinPipe()
		if err != nil {
			fmt.Printf("    Error decrypting lock key (get stdin): %s\n", err.Error)
			return err
		}
		go func() {
			defer inPipe.Close()
			inPipe.Write(lockkeyData)
			//TODO: catch error from this somehow.  maybe w/ channel?
		}()

		aesPadKeyIv, err := cmd.Output()
		if err != nil {
			fmt.Printf("    Error decrypting lock key (command execution): %s\n", err.Error)
			return err
		}

		actualDecryptedLen := len(aesPadKeyIv)
		if actualDecryptedLen < DecryptedLockKeyLength {
			errString := fmt.Sprintf("Error, decrypted padding, key, and iv length (%d) are not correct (expected >= %d)",
				len(aesPadKeyIv), DecryptedLockKeyLength)
			return errors.New(errString)
		}
		
		padding := aesPadKeyIv[actualDecryptedLen - PadLength - AesLength - IvLength]
		aesKeyBytes := aesPadKeyIv[actualDecryptedLen - AesLength - IvLength : actualDecryptedLen - IvLength]
		iv := aesPadKeyIv[actualDecryptedLen - IvLength : actualDecryptedLen]
		aesKey, err := aes.NewCipher(aesKeyBytes)
		if err != nil {
			return err
		}
		//decrypt payload
		if len(encryptedData)%aes.BlockSize != 0 {
			return errors.New("Encrypted payload is not a multiple of the block size")
		}
		decrypter := cipher.NewCBCDecrypter(aesKey, iv)
		clearFilePadded := make([]byte, len(encryptedData))
		unpaddedLen := len(encryptedData) - int(padding)
		decrypter.CryptBlocks(clearFilePadded, encryptedData)
		clearFile := clearFilePadded[0:unpaddedLen]
		unencryptedReader = bytes.NewReader(clearFile)
		fmt.Println("  Decryption complete.")
	}	
	//handle decrypted (or unencrypted) payload
	archive, err := gzip.NewReader(unencryptedReader)
	if err != nil {
		return err
	}
	defer archive.Close()
	fmt.Println("  Unpacking data payload...")
	tarReader = tar.NewReader(archive)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		fmt.Printf("    Examining %s\n", header.Name)
		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		invalid, _ := regexp.MatchString(`^.*\.\.\/.*$`, path)
		if invalid == false {
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
		} else {
			return errors.New("Invalid data payload tarball")
		}
	}
	fmt.Println("  Data payload unpacking complete.")
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

// Make a persistent backup of source
func CreatePersistentBackup(source io.Reader, target_name string, target_dir string) error {
	tgt_pathname := filepath.Join(target_dir, target_name)
	os.Mkdir(target_dir, os.FileMode(0755))
	file, err := os.OpenFile(tgt_pathname, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, source)
	if err != nil {
		return err
	}
	return nil
}
