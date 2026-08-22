package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

var (
	ErrBackupInProgress     = errors.New("já existe um backup em andamento")
	ErrInvalidConfiguration = errors.New("configuração de backup inválida ou incompleta")
	ErrPgDumpFailed         = errors.New("falha na execução do pg_dump")
	ErrValidationFailed     = errors.New("falha na validação do arquivo de backup")
	ErrUploadFailed         = errors.New("falha no envio para o Cloudflare R2")
	ErrPresignFailed        = errors.New("falha na geração da URL assinada")
)

type BackupConfig struct {
	DatabaseURL         string
	R2AccountID         string
	R2AccessKeyID       string
	R2SecretAccessKey   string
	R2BackupBucket      string
	R2BackupPrefix      string
	BackupURLExpiration time.Duration
	BackupTimeout       time.Duration
}

type BackupResult struct {
	Filename    string
	ObjectKey   string
	Size        int64
	SHA256      string
	DownloadURL string
	ExpiresAt   time.Time
	Duration    time.Duration
}

// DBDumper define a interface de geração e validação de dump para PostgreSQL.
type DBDumper interface {
	Dump(ctx context.Context, destFile string) error
	Validate(ctx context.Context, dumpFile string) error
}

// ObjectStorage define a interface de envio e assinatura de URLs no Cloudflare R2 / S3.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64) error
	PresignGet(ctx context.Context, key string, expiration time.Duration) (string, error)
}

type PostgresDumper struct {
	databaseURL string
}

func NewPostgresDumper(databaseURL string) *PostgresDumper {
	return &PostgresDumper{databaseURL: databaseURL}
}

func (d *PostgresDumper) Dump(ctx context.Context, destFile string) error {
	if d.databaseURL == "" {
		return fmt.Errorf("%w: DATABASE_URL não configurada", ErrInvalidConfiguration)
	}

	args := []string{
		"--format=custom",
		"--compress=6",
		"--no-owner",
		"--no-privileges",
		"--exclude-table-data=public.channel_events",
		fmt.Sprintf("--file=%s", destFile),
		fmt.Sprintf("--dbname=%s", d.databaseURL),
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Log seguro sem vazar a connection string
		logger.Error("BACKUP", "Falha na execução do pg_dump: %v", err)
		return fmt.Errorf("%w: %v", ErrPgDumpFailed, err)
	}

	return nil
}

func (d *PostgresDumper) Validate(ctx context.Context, dumpFile string) error {
	stat, err := os.Stat(dumpFile)
	if err != nil {
		return fmt.Errorf("%w: arquivo de dump não encontrado: %v", ErrValidationFailed, err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("%w: arquivo de dump está vazio", ErrValidationFailed)
	}

	cmd := exec.CommandContext(ctx, "pg_restore", "--list", dumpFile)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Error("BACKUP", "Falha na validação do pg_restore: %v", err)
		return fmt.Errorf("%w: pg_restore --list falhou: %v", ErrValidationFailed, err)
	}

	toc := stdout.String()
	for _, line := range strings.Split(toc, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "table data") && strings.Contains(lower, "channel_events") {
			logger.Error("BACKUP", "Validação falhou: dump contém dados proibidos de channel_events: %s", line)
			return fmt.Errorf("%w: dump contém dados da tabela channel_events", ErrValidationFailed)
		}
	}

	return nil
}

type R2Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func NewR2Storage(ctx context.Context, accountID, accessKeyID, secretAccessKey, bucket string) (*R2Storage, error) {
	if accountID == "" || accessKeyID == "" || secretAccessKey == "" || bucket == "" {
		return nil, ErrInvalidConfiguration
	}

	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar configuração AWS/R2: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	presignClient := s3.NewPresignClient(client, func(o *s3.PresignOptions) {
		o.ClientOptions = append(o.ClientOptions, func(so *s3.Options) {
			so.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			so.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		})
	})

	return &R2Storage{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

func (r *R2Storage) Upload(ctx context.Context, key string, body io.Reader, size int64) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	return nil
}

func (r *R2Storage) PresignGet(ctx context.Context, key string, expiration time.Duration) (string, error) {
	req, err := r.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPresignFailed, err)
	}
	return req.URL, nil
}

type BackupService struct {
	cfg       BackupConfig
	dumper    DBDumper
	storage   ObjectStorage
	isRunning atomic.Bool
}

func NewBackupService(cfg BackupConfig) (*BackupService, error) {
	dumper := NewPostgresDumper(cfg.DatabaseURL)
	storage, err := NewR2Storage(context.Background(), cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2BackupBucket)
	if err != nil {
		logger.Warn("BACKUP", "Aviso: Cloudflare R2 não inicializado (%v). O serviço estará indisponível até configuração completa.", err)
		return &BackupService{
			cfg:    cfg,
			dumper: dumper,
		}, nil
	}

	return &BackupService{
		cfg:     cfg,
		dumper:  dumper,
		storage: storage,
	}, nil
}

func NewBackupServiceWithCustom(cfg BackupConfig, dumper DBDumper, storage ObjectStorage) *BackupService {
	return &BackupService{
		cfg:     cfg,
		dumper:  dumper,
		storage: storage,
	}
}

func (s *BackupService) IsConfigured() bool {
	return s.storage != nil && s.cfg.DatabaseURL != ""
}

func (s *BackupService) GetURLExpirationString() string {
	if s.cfg.BackupURLExpiration > 0 {
		return s.cfg.BackupURLExpiration.String()
	}
	return "15m"
}

func (s *BackupService) PerformBackup(ctx context.Context) (*BackupResult, error) {
	if !s.isRunning.CompareAndSwap(false, true) {
		return nil, ErrBackupInProgress
	}
	defer s.isRunning.Store(false)

	startTime := time.Now()

	if s.dumper == nil || s.storage == nil {
		return nil, ErrInvalidConfiguration
	}

	tempDir, err := os.MkdirTemp("", "freddybot-backup-*")
	if err != nil {
		logger.Error("BACKUP", "Erro ao criar diretório temporário: %v", err)
		return nil, fmt.Errorf("criar diretório temporário: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	now := time.Now().UTC()
	filename := fmt.Sprintf("legendasbot-%s.dmp", now.Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(tempDir, filename)

	timeout := s.cfg.BackupTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	backupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logger.Info("BACKUP", "Iniciando geração de backup: %s", filename)

	// 1. Executar pg_dump
	if err := s.dumper.Dump(backupCtx, filePath); err != nil {
		logger.Error("BACKUP", "Falha na etapa pg_dump: %v", err)
		return nil, err
	}

	// 2. Validar dump
	if err := s.dumper.Validate(backupCtx, filePath); err != nil {
		logger.Error("BACKUP", "Falha na validação do dump: %v", err)
		return nil, err
	}

	// 3. Obter tamanho e SHA-256
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("abrir dump para leitura: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("obter tamanho do dump: %w", err)
	}
	size := stat.Size()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("calcular sha256 do dump: %w", err)
	}
	sha256Hex := hex.EncodeToString(hasher.Sum(nil))

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek arquivo dump: %w", err)
	}

	// 4. Upload para o Cloudflare R2
	prefix := strings.Trim(s.cfg.R2BackupPrefix, "/")
	var objectKey string
	if prefix != "" {
		objectKey = fmt.Sprintf("%s/%s", prefix, filename)
	} else {
		objectKey = filename
	}

	logger.Info("BACKUP", "Enviando backup para Cloudflare R2: %s (%d bytes)", objectKey, size)
	if err := s.storage.Upload(backupCtx, objectKey, file, size); err != nil {
		logger.Error("BACKUP", "Falha no upload para R2: %v", err)
		return nil, err
	}

	// 5. Gerar Presigned GET URL
	expiration := s.cfg.BackupURLExpiration
	if expiration <= 0 {
		expiration = 15 * time.Minute
	}
	presignedURL, err := s.storage.PresignGet(backupCtx, objectKey, expiration)
	if err != nil {
		logger.Error("BACKUP", "Falha ao gerar presigned URL: %v", err)
		return nil, err
	}

	duration := time.Since(startTime)
	logger.Info("BACKUP", "Backup concluído com sucesso em %v. SHA256: %s", duration, sha256Hex)

	return &BackupResult{
		Filename:    filename,
		ObjectKey:   objectKey,
		Size:        size,
		SHA256:      sha256Hex,
		DownloadURL: presignedURL,
		ExpiresAt:   time.Now().Add(expiration),
		Duration:    duration,
	}, nil
}
