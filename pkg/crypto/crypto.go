package crypto

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const EncryptedPrefix = "enc:v1:"

// deriveKey gera uma chave AES de 32 bytes (256 bits) a partir de qualquer segredo string.
func deriveKey(secretKey string) []byte {
	hash := sha256.Sum256([]byte(secretKey))
	return hash[:]
}

// CompressAndEncrypt compacta os dados com GZIP e cifra usando AES-256-GCM.
func CompressAndEncrypt(plaintext []byte, secretKey string) (string, error) {
	if len(plaintext) == 0 {
		return "", nil
	}

	// 1. Compactação GZIP
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(plaintext); err != nil {
		return "", fmt.Errorf("erro na compactação gzip: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("erro no fechamento gzip: %w", err)
	}
	compressed := buf.Bytes()

	// 2. Cifragem AES-256-GCM
	key := deriveKey(secretKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("erro ao criar cipher aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("erro ao criar gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("erro ao gerar nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, compressed, nil)

	// 3. Codificação Base64 com prefixo de versão
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// DecryptAndDecompress decifra os dados com AES-256-GCM e descompacta via GZIP.
func DecryptAndDecompress(encodedStr string, secretKey string) ([]byte, error) {
	if encodedStr == "" {
		return []byte{}, nil
	}

	// Se não possuir o prefixo "enc:v1:", trata como dado legado em texto claro (JSON antigo)
	if !strings.HasPrefix(encodedStr, EncryptedPrefix) {
		return []byte(encodedStr), nil
	}

	rawBase64 := strings.TrimPrefix(encodedStr, EncryptedPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	key := deriveKey(secretKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cipher aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("tamanho de ciphertext invalido")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	compressed, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao decifrar dados: %w", err)
	}

	// Descompactação GZIP
	gzReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados descomprimidos: %w", err)
	}

	return decompressed, nil
}
