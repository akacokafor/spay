package spay

import (
	"crypto/cipher"
	"crypto/des"
	b64 "encoding/base64"
	"errors"
	"fmt"
)

func TripleDESCBCEncrypt(input string, encryptionKey, encryptionvector []byte) (string, error) {
	key := encryptionKey
	iv := encryptionvector
	pad := des.BlockSize - len(input)%des.BlockSize
	if pad == 0 {
		pad = des.BlockSize
	}
	data := make([]byte, len(input)+pad)
	copy(data, input)
	for i := len(input); i < len(input)+pad; i++ {
		data[i] = byte(pad)
	}
	cb, err := des.NewTripleDESCipher(key)
	if err != nil {
		return "", fmt.Errorf("new tripledes cipher: %w", err)
	}

	mode := cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(data, data)
	return b64.StdEncoding.EncodeToString(data), nil
}

func TripleDESCBCDecrypt(payload string, encryptionKey, encryptionvector []byte) (string, error) {
	key := encryptionKey
	iv := encryptionvector
	ciphertext, err := b64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}

	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < des.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	if len(ciphertext)%des.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of the block size")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	return string(plaintext), nil
}
