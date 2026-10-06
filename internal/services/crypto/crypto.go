// Package crypto — AES-256-GCM шифрование значений настроек.
// Ключ шифрования хранится в файле .secret в директории БД,
// создаётся при первом старте с правами 0600.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// KeySize — длина ключа AES-256 в байтах.
const KeySize = 32

// Файл с ключом шифрования в директории БД.
const (
	secretFileName = ".secret"
	secretFileMode = 0o600
	secretDirMode  = 0o755
)

// LoadOrCreateKey читает ключ шифрования из файла .secret в dir;
// при отсутствии файла генерирует KeySize случайных байт и сохраняет
// их с правами 0600.
func LoadOrCreateKey(dir string) ([]byte, error) {
	path := filepath.Join(dir, secretFileName)
	key, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(key) != KeySize {
			return nil, fmt.Errorf("secret key file %s must contain exactly %d bytes", path, KeySize)
		}
		return key, nil
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("read secret key file: %w", err)
	}
	if err := os.MkdirAll(dir, secretDirMode); err != nil {
		return nil, fmt.Errorf("create secret key dir: %w", err)
	}
	key = make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}
	if err := os.WriteFile(path, key, secretFileMode); err != nil {
		return nil, fmt.Errorf("write secret key file: %w", err)
	}
	return key, nil
}

// Cipher шифрует и расшифровывает строки алгоритмом AES-256-GCM;
// случайный nonce хранится префиксом шифртекста, результат — base64.
type Cipher struct {
	aead cipher.AEAD
}

// New создаёт Cipher из ключа длиной KeySize байт.
func New(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("encryption key must be exactly %d bytes", KeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt шифрует строку и возвращает base64(nonce + шифртекст).
func (c *Cipher) Encrypt(plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt расшифровывает строку, созданную Encrypt.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	if len(raw) < c.aead.NonceSize() {
		return "", fmt.Errorf("ciphertext is too short")
	}
	nonce, sealed := raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():]
	plain, err := c.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plain), nil
}
