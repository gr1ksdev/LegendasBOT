package crypto_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leirbagxis/FreddyBot/pkg/crypto"
)

func TestCompressAndEncrypt(t *testing.T) {
	secretKey := "super-secret-key-freddybot-12345"
	originalJSON := []byte(`{"media_type":"photo","media_file_id":"AgACAgEAAxkBAAIBZ","title":"Oferta Especial de Lançamento","body":"Confira agora os novos produtos disponíveis na loja oficial do canal! Aproveite os descontos por tempo limitado.","footer":"@FreddyBot Oficial","buttons":[{"text":"Comprar","url":"https://loja.com"},{"text":"Suporte","url":"https://t.me/suporte"}],"reactions":"👍,🔥,❤️"}`)

	encrypted, err := crypto.CompressAndEncrypt(originalJSON, secretKey)
	if err != nil {
		t.Fatalf("Erro ao compactar e criptografar: %v", err)
	}

	if !strings.HasPrefix(encrypted, crypto.EncryptedPrefix) {
		t.Errorf("Esperava prefixo %s, obteve: %s", crypto.EncryptedPrefix, encrypted)
	}

	t.Logf("Tamanho original: %d bytes | Criptografado+Compactado: %d bytes", len(originalJSON), len(encrypted))

	decrypted, err := crypto.DecryptAndDecompress(encrypted, secretKey)
	if err != nil {
		t.Fatalf("Erro ao decifrar e descompactar: %v", err)
	}

	if !bytes.Equal(originalJSON, decrypted) {
		t.Errorf("Esperava %s, obteve: %s", string(originalJSON), string(decrypted))
	}
}

func TestLegacyPlainTextData(t *testing.T) {
	secretKey := "super-secret-key-freddybot-12345"
	legacyJSON := `{"media_type":"text","body":"Rascunho antigo salvo em texto puro sem criptografia"}`

	decrypted, err := crypto.DecryptAndDecompress(legacyJSON, secretKey)
	if err != nil {
		t.Fatalf("Erro ao ler dado legado: %v", err)
	}

	if string(decrypted) != legacyJSON {
		t.Errorf("Esperava %s, obteve: %s", legacyJSON, string(decrypted))
	}
}
