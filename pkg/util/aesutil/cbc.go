package aesutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// Code is from https://golang.org/src/crypto/cipher/example_test.go

func EncryptCBC(content, key string) []byte {
	// Load your secret key from a safe place and reuse it across multiple
	// NewCipher calls. (Obviously don't use this example key for anything
	// real.) If you want to convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	kb, _ := hex.DecodeString(key)
	kb = formatKey(kb)
	block, _ := aes.NewCipher(kb)
	data := []byte(content)
	data = pkcs5Padding(data, aes.BlockSize)

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize + len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], data)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	return ciphertext
}

func DecryptCBC(ciphertext []byte, key string) []byte {
	// Load your secret key from a safe place and reuse it across multiple
	// NewCipher calls. (Obviously don't use this example key for anything
	// real.) If you want to convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	kb, _ := hex.DecodeString(key)
	kb = formatKey(kb)
	block, _ := aes.NewCipher(kb)

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point. For an example, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. However, it's
	// critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.

	return pkcs5UnPadding(ciphertext)
}

func pkcs5Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data) % blockSize
	pad := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, pad...)
}

func pkcs5UnPadding(data []byte) []byte {
	length := len(data)
	pad := int(data[length - 1])
	return data[:(length - pad)]
}

func formatKey(key []byte) []byte {
	if len(key) >= 32 {
		return key[:32]
	} else {
		padding := 32 - len(key)
		pad := bytes.Repeat([]byte{byte(padding)}, padding)
		return append(key, pad...)
	}
}
