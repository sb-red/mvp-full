package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
)

// StorageService interface for code file storage
type StorageService interface {
	SaveCode(ctx context.Context, key string, code string) error
	GetCode(ctx context.Context, key string) (string, error)
	DeleteCode(ctx context.Context, key string) error
}

// LocalStorageService implements StorageService using local filesystem
type LocalStorageService struct {
	basePath string
}

func NewLocalStorageService(basePath string) (*LocalStorageService, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &LocalStorageService{basePath: basePath}, nil
}

func (s *LocalStorageService) SaveCode(ctx context.Context, key string, code string) error {
	fullPath := filepath.Join(s.basePath, key)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(code), 0644)
}

func (s *LocalStorageService) GetCode(ctx context.Context, key string) (string, error) {
	fullPath := filepath.Join(s.basePath, key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *LocalStorageService) DeleteCode(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)
	return os.Remove(fullPath)
}

// S3StorageService implements StorageService using AWS S3
type S3StorageService struct {
	client *s3.Client
	bucket string
}

func NewS3StorageService(bucket string) (*S3StorageService, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	// Instrument AWS SDK v2 with X-Ray for automatic S3 operation tracing
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	client := s3.NewFromConfig(cfg)
	return &S3StorageService{client: client, bucket: bucket}, nil
}

func (s *S3StorageService) SaveCode(ctx context.Context, key string, code string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(code),
		ContentType: aws.String("text/plain"),
	})
	return err
}

func (s *S3StorageService) GetCode(ctx context.Context, key string) (string, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *S3StorageService) DeleteCode(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// NewStorageService creates appropriate storage service based on environment
func NewStorageService(storageType, pathOrBucket string) (StorageService, error) {
	switch storageType {
	case "s3":
		return NewS3StorageService(pathOrBucket)
	case "local":
		return NewLocalStorageService(pathOrBucket)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}
}

// GenerateCodeKey generates a unique key for storing function code
func GenerateCodeKey(functionID int64, runtime string) string {
	var ext string
	switch runtime {
	case "python3.11", "python", "pypy3":
		ext = ".py"
	case "javascript", "node", "nodejs18":
		ext = ".js"
	case "java11", "java17", "java21":
		ext = ".java"
	case "swift":
		ext = ".swift"
	case "kotlin":
		ext = ".kt"
	default:
		ext = ".txt" // fallback for unknown runtimes
	}
	return fmt.Sprintf("code/functions/func_%d%s", functionID, ext)
}
