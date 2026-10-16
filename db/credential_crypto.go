package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

const credentialPrefix = "tyan:v1:"

func credentialCipher() (cipher.AEAD, error) {
	key, err := base64.StdEncoding.DecodeString(os.Getenv("TYAN_CREDENTIAL_KEY"))
	if err != nil || len(key) != 32 {
		return nil, errors.New("set TYAN_CREDENTIAL_KEY to a base64-encoded 32-byte key before storing or reading passwords")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func sealTaskPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	aead, err := credentialCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return credentialPrefix + base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(password), []byte(credentialPrefix))), nil
}

func openTaskPassword(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, credentialPrefix) {
		return "", errors.New("unencrypted legacy task password refused; remove and re-enter this credential")
	}
	aead, err := credentialCipher()
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, credentialPrefix))
	if err != nil || len(data) < aead.NonceSize() {
		return "", errors.New("invalid encrypted task password")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(credentialPrefix))
	if err != nil {
		return "", errors.New("cannot decrypt task password; verify TYAN_CREDENTIAL_KEY")
	}
	return string(plain), nil
}
