// Package encryption fornece metodos para criptografar e descriptografar
// sessoes MTProto antes de persisti-las no banco de dados.
//
// Usa AES-256-GCM com chave derivada da MTPROTO_ENCRYPTION_KEY definida em
// variavel de ambiente (com fallback retrocompativel para SECRET_KEY).
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/leirbagxis/FreddyBot/pkg/config"
)

// GetKey retorna os 256 bits de chave derivadas de MTPROTO_ENCRYPTION_KEY (ou SecretKey fallback).
func GetKey() []byte {
	return DeriveKey(config.MTProtoEncryptionKey)
}

// DeriveKey deriva uma chave de 256 bits (32 bytes) a partir da chave
// fornecida usando SHA-256. Isso garante que qualquer tamanho de chave
// seja aceitavel e produz uma chave AES-256 valida.
func DeriveKey(key string) []byte {
	h := sha256.Sum256([]byte(key))
	return h[:]
}

// Encrypt criptografa dados usando AES-256-GCM.
// Retorna o ciphertext em formato hexadecimal (nonce + ciphertext + auth tag).
func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt descriptografa dados que foram criptografados com Encrypt.
// Recebe o ciphertext em formato hexadecimal.
func Decrypt(cipherHex string, key []byte) ([]byte, error) {
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
