package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

type Encrypter struct {
	key []byte
}

func NewEncrypter() *Encrypter {
	key := os.Getenv("BOMBS_MESSAGE_ENCRYPTION_KEY")
	if len(key) < 32 {
		panic("BOMBS_MESSAGE_ENCRYPTION_KEY must be longer than 32 characters.")
	}
	return &Encrypter{key: []byte(key[:32])}
}

// func main() {
// 	plaintext := []byte("Hello, Go encryption!")
// 	key := []byte("32-byte-key-for-AES-256-!!!!!!!!") // 32 bytes for AES-256

// 	// Encrypt
// 	ciphertext, err := encryptAES(plaintext, key)
// 	if err != nil {
// 		fmt.Println("Encryption error:", err)
// 		return
// 	}
// 	fmt.Printf("Ciphertext (base64): %s\n", base64.StdEncoding.EncodeToString(ciphertext))

// 	// Decrypt
// 	decrypted, err := decryptAES(ciphertext, key)
// 	if err != nil {
// 		fmt.Println("Decryption error:", err)
// 		return
// 	}
// 	fmt.Printf("Decrypted: %s\n", decrypted)
// }

func (e *Encrypter) Encrypt(text string) ([]byte, error) {
	plaintext := []byte(text)
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	// Pad plaintext to block size
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := append(plaintext, bytes.Repeat([]byte{byte(padding)}, padding)...)

	ciphertext := make([]byte, aes.BlockSize+len(padtext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], padtext)

	return ciphertext, nil
}

func (e *Encrypter) Decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Unpad
	padding := int(ciphertext[len(ciphertext)-1])
	textbytes := ciphertext[:len(ciphertext)-padding]

	// Convert to string
	text := string(textbytes)

	return text, nil
}
