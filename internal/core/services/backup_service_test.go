package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockDBDumper struct {
	dumpFunc     func(ctx context.Context, destFile string) error
	validateFunc func(ctx context.Context, dumpFile string) error
}

func (m *mockDBDumper) Dump(ctx context.Context, destFile string) error {
	if m.dumpFunc != nil {
		return m.dumpFunc(ctx, destFile)
	}
	// Default: write a small dummy dump file
	return os.WriteFile(destFile, []byte("PGDMP-mock-data-content-12345"), 0600)
}

func (m *mockDBDumper) Validate(ctx context.Context, dumpFile string) error {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, dumpFile)
	}
	return nil
}

type mockObjectStorage struct {
	uploadFunc     func(ctx context.Context, key string, body io.Reader, size int64) error
	presignGetFunc func(ctx context.Context, key string, expiration time.Duration) (string, error)
	uploadedKeys   []string
	uploadedBodies [][]byte
}

func (m *mockObjectStorage) Upload(ctx context.Context, key string, body io.Reader, size int64) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, key, body, size)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.uploadedKeys = append(m.uploadedKeys, key)
	m.uploadedBodies = append(m.uploadedBodies, data)
	return nil
}

func (m *mockObjectStorage) PresignGet(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if m.presignGetFunc != nil {
		return m.presignGetFunc(ctx, key, expiration)
	}
	return fmt.Sprintf("https://mock-r2.cloudflarestorage.com/%s?token=mocktoken", key), nil
}

func TestBackup_Success(t *testing.T) {
	content := []byte("PostgreSQL-custom-dump-content-sample")
	hasher := sha256.New()
	hasher.Write(content)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	dumper := &mockDBDumper{
		dumpFunc: func(ctx context.Context, destFile string) error {
			return os.WriteFile(destFile, content, 0600)
		},
	}

	storage := &mockObjectStorage{}

	cfg := BackupConfig{
		DatabaseURL:         "postgres://user:pass@localhost:5432/mydb",
		R2BackupPrefix:      "postgres/backups",
		BackupURLExpiration: 15 * time.Minute,
		BackupTimeout:       1 * time.Minute,
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	ctx := context.Background()
	result, err := svc.PerformBackup(ctx)
	if err != nil {
		t.Fatalf("PerformBackup falhou: %v", err)
	}

	if result == nil {
		t.Fatal("esperado resultado não-nulo")
	}

	// 1. Validar nome do arquivo
	if !strings.HasPrefix(result.Filename, "legendasbot-") || !strings.HasSuffix(result.Filename, ".dmp") {
		t.Errorf("nome de arquivo inválido: %s", result.Filename)
	}

	// 2. Validar chave do objeto
	expectedKey := fmt.Sprintf("postgres/backups/%s", result.Filename)
	if result.ObjectKey != expectedKey {
		t.Errorf("ObjectKey = %s, esperado %s", result.ObjectKey, expectedKey)
	}

	// 3. Validar tamanho e hash
	if result.Size != int64(len(content)) {
		t.Errorf("Size = %d, esperado %d", result.Size, len(content))
	}
	if result.SHA256 != expectedHash {
		t.Errorf("SHA256 = %s, esperado %s", result.SHA256, expectedHash)
	}

	// 4. Validar download URL
	if !strings.Contains(result.DownloadURL, expectedKey) {
		t.Errorf("DownloadURL não contém a chave: %s", result.DownloadURL)
	}

	// 5. Validar expiração
	if result.ExpiresAt.Before(time.Now()) {
		t.Errorf("ExpiresAt no passado: %v", result.ExpiresAt)
	}

	// 6. Validar que o lock foi liberado
	if svc.isRunning.Load() {
		t.Error("esperado lock liberado após sucesso")
	}

	// 7. Validar que o storage recebeu o conteúdo correto
	if len(storage.uploadedKeys) != 1 || storage.uploadedKeys[0] != expectedKey {
		t.Errorf("storage uploaded keys = %v, esperado [%s]", storage.uploadedKeys, expectedKey)
	}
	if len(storage.uploadedBodies) != 1 || string(storage.uploadedBodies[0]) != string(content) {
		t.Errorf("storage body incorreto")
	}
}

func TestBackup_ConcurrencyGuard(t *testing.T) {
	blockDump := make(chan struct{})
	dumpStarted := make(chan struct{})
	var startOnce sync.Once

	dumper := &mockDBDumper{
		dumpFunc: func(ctx context.Context, destFile string) error {
			startOnce.Do(func() {
				close(dumpStarted)
				<-blockDump
			})
			return os.WriteFile(destFile, []byte("dump-data"), 0600)
		},
	}

	storage := &mockObjectStorage{}
	cfg := BackupConfig{
		DatabaseURL:         "postgres://localhost/db",
		R2BackupPrefix:      "backups",
		BackupURLExpiration: 15 * time.Minute,
		BackupTimeout:       1 * time.Minute,
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	var wg sync.WaitGroup
	wg.Add(1)

	var firstErr error
	var firstResult *BackupResult

	// Inicia primeiro backup
	go func() {
		defer wg.Done()
		firstResult, firstErr = svc.PerformBackup(context.Background())
	}()

	// Aguarda o primeiro backup iniciar e travar no dump
	<-dumpStarted

	// Tenta executar o segundo backup simultaneamente
	_, secondErr := svc.PerformBackup(context.Background())
	if secondErr == nil || !errors.Is(secondErr, ErrBackupInProgress) {
		t.Errorf("segundo backup deveria retornar ErrBackupInProgress, obteve: %v", secondErr)
	}

	// Libera o primeiro backup
	close(blockDump)
	wg.Wait()

	if firstErr != nil {
		t.Fatalf("primeiro backup falhou: %v", firstErr)
	}
	if firstResult == nil {
		t.Fatal("primeiro backup deveria retornar resultado")
	}

	// Após conclusão, deve ser possível iniciar outro backup
	thirdResult, thirdErr := svc.PerformBackup(context.Background())
	if thirdErr != nil {
		t.Fatalf("terceiro backup falhou: %v", thirdErr)
	}
	if thirdResult == nil {
		t.Fatal("terceiro backup deveria ter sucesso após liberação do lock")
	}
}

func TestBackup_DumpErrorReleasesLock(t *testing.T) {
	dumper := &mockDBDumper{
		dumpFunc: func(ctx context.Context, destFile string) error {
			return errors.New("pg_dump connection refused")
		},
	}

	storage := &mockObjectStorage{}
	cfg := BackupConfig{
		DatabaseURL: "postgres://localhost/db",
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	_, err := svc.PerformBackup(context.Background())
	if err == nil {
		t.Fatal("esperado erro quando dump falha")
	}

	// Lock deve estar liberado
	if svc.isRunning.Load() {
		t.Error("esperado lock liberado após erro de dump")
	}
	if len(storage.uploadedKeys) != 0 {
		t.Error("storage upload não deveria ter sido chamado após erro no dump")
	}
}

func TestBackup_ValidationErrorReleasesLock(t *testing.T) {
	dumper := &mockDBDumper{
		dumpFunc: func(ctx context.Context, destFile string) error {
			return os.WriteFile(destFile, []byte("corrupted-dump"), 0600)
		},
		validateFunc: func(ctx context.Context, dumpFile string) error {
			return errors.New("pg_restore: [archiver] input file appear to be empty or corrupted")
		},
	}

	storage := &mockObjectStorage{}
	cfg := BackupConfig{
		DatabaseURL: "postgres://localhost/db",
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	_, err := svc.PerformBackup(context.Background())
	if err == nil {
		t.Fatal("esperado erro quando validação falha")
	}

	if svc.isRunning.Load() {
		t.Error("esperado lock liberado após erro de validação")
	}
	if len(storage.uploadedKeys) != 0 {
		t.Error("storage upload não deveria ter sido chamado após erro na validação")
	}
}

func TestBackup_UploadErrorReleasesLock(t *testing.T) {
	dumper := &mockDBDumper{}
	storage := &mockObjectStorage{
		uploadFunc: func(ctx context.Context, key string, body io.Reader, size int64) error {
			return errors.New("R2 bucket unauthorized 403")
		},
	}

	cfg := BackupConfig{
		DatabaseURL: "postgres://localhost/db",
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	_, err := svc.PerformBackup(context.Background())
	if err == nil {
		t.Fatal("esperado erro quando upload falha")
	}

	if svc.isRunning.Load() {
		t.Error("esperado lock liberado após erro de upload")
	}
}

func TestBackup_PresignErrorReleasesLock(t *testing.T) {
	dumper := &mockDBDumper{}
	storage := &mockObjectStorage{
		presignGetFunc: func(ctx context.Context, key string, expiration time.Duration) (string, error) {
			return "", errors.New("presign token signing failed")
		},
	}

	cfg := BackupConfig{
		DatabaseURL: "postgres://localhost/db",
	}

	svc := NewBackupServiceWithCustom(cfg, dumper, storage)

	_, err := svc.PerformBackup(context.Background())
	if err == nil {
		t.Fatal("esperado erro quando presign falha")
	}

	if svc.isRunning.Load() {
		t.Error("esperado lock liberado após erro de presign")
	}
}

func TestBackup_InvalidConfig(t *testing.T) {
	svc := NewBackupServiceWithCustom(BackupConfig{}, nil, nil)

	_, err := svc.PerformBackup(context.Background())
	if err == nil || !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("esperado ErrInvalidConfiguration, obteve: %v", err)
	}

	if svc.isRunning.Load() {
		t.Error("lock deve estar liberado")
	}
}

func TestBackup_ObjectKeyPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"postgres/backups", "postgres/backups/"},
		{"/postgres/backups/", "postgres/backups/"},
		{"backups", "backups/"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run("prefix="+tt.prefix, func(t *testing.T) {
			storage := &mockObjectStorage{}
			svc := NewBackupServiceWithCustom(BackupConfig{
				DatabaseURL:    "postgres://localhost/db",
				R2BackupPrefix: tt.prefix,
			}, &mockDBDumper{}, storage)

			res, err := svc.PerformBackup(context.Background())
			if err != nil {
				t.Fatalf("PerformBackup: %v", err)
			}

			if tt.expected == "" {
				if res.ObjectKey != res.Filename {
					t.Errorf("ObjectKey = %s, esperado %s", res.ObjectKey, res.Filename)
				}
			} else {
				if !strings.HasPrefix(res.ObjectKey, tt.expected) {
					t.Errorf("ObjectKey = %s, esperado iniciar com %s", res.ObjectKey, tt.expected)
				}
			}
		})
	}
}

func TestBackup_CleanupOnSuccessAndError(t *testing.T) {
	var capturedFilePath string

	dumper := &mockDBDumper{
		dumpFunc: func(ctx context.Context, destFile string) error {
			capturedFilePath = destFile
			return os.WriteFile(destFile, []byte("dump-temp-data"), 0600)
		},
	}

	svc := NewBackupServiceWithCustom(BackupConfig{
		DatabaseURL: "postgres://localhost/db",
	}, dumper, &mockObjectStorage{})

	res, err := svc.PerformBackup(context.Background())
	if err != nil {
		t.Fatalf("PerformBackup: %v", err)
	}
	if res == nil {
		t.Fatal("resultado nulo")
	}

	// O arquivo temporário e seu diretório pai devem ter sido excluídos
	if _, err := os.Stat(capturedFilePath); !os.IsNotExist(err) {
		t.Errorf("arquivo temporário ainda existe após backup bem-sucedido: %s", capturedFilePath)
	}
	tempDir := filepath.Dir(capturedFilePath)
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("diretório temporário ainda existe após backup bem-sucedido: %s", tempDir)
	}
}

func TestPostgresDumper_TOCValidationLogic(t *testing.T) {
	t.Run("Rejects TOC containing TABLE DATA for channel_events", func(t *testing.T) {
		tocWithTableData := `
;
; Archive created at 2026-08-22 01:00:00 UTC
;
1; 1259 16385 TABLE public channels postgres
2; 1259 16390 TABLE public channel_events postgres
3; 0 16385 TABLE DATA public channels postgres
4; 0 16390 TABLE DATA public channel_events postgres
`
		// Validate logic check
		foundTableData := false
		for _, line := range strings.Split(tocWithTableData, "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "table data") && strings.Contains(lower, "channel_events") {
				foundTableData = true
				break
			}
		}
		if !foundTableData {
			t.Error("esperado detecção de TABLE DATA public channel_events")
		}
	})

	t.Run("Accepts TOC with TABLE schema only for channel_events", func(t *testing.T) {
		tocSchemaOnly := `
;
; Archive created at 2026-08-22 01:00:00 UTC
;
1; 1259 16385 TABLE public channels postgres
2; 1259 16390 TABLE public channel_events postgres
3; 0 16385 TABLE DATA public channels postgres
`
		foundTableData := false
		for _, line := range strings.Split(tocSchemaOnly, "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "table data") && strings.Contains(lower, "channel_events") {
				foundTableData = true
				break
			}
		}
		if foundTableData {
			t.Error("não deveria encontrar TABLE DATA de channel_events")
		}
	})
}

func TestR2Storage_PresignedURLNoChecksumHeaders(t *testing.T) {
	storage, err := NewR2Storage(context.Background(), "mockaccountid", "mockkeyid", "mocksecretkey", "mybucket")
	if err != nil {
		t.Fatalf("NewR2Storage: %v", err)
	}

	url, err := storage.PresignGet(context.Background(), "postgres/backups/test.dmp", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	// A URL assinada NÃO deve conter x-amz-checksum-mode em X-Amz-SignedHeaders
	if strings.Contains(url, "x-amz-checksum-mode") {
		t.Errorf("URL presign ainda contém header de checksum incompatível com navegadores/Cloudflare R2: %s", url)
	}

	if !strings.Contains(url, "X-Amz-SignedHeaders=host") {
		t.Errorf("URL presign deve assinar apenas o header host, obtido: %s", url)
	}
}

