package destinations

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
)

// S3Destination implements backup destination for S3-compatible storage
type S3Destination struct {
	client *s3.Client
	bucket string
	prefix string // Optional prefix for all keys
}

// NewS3Destination creates a new S3 backup destination
func NewS3Destination(cfg *btypes.S3Config) (*S3Destination, error) {
	// Create custom AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint if provided
	var client *s3.Client
	if cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO and other S3-compatible services
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	return &S3Destination{
		client: client,
		bucket: cfg.Bucket,
		prefix: "", // Could be configured if needed
	}, nil
}

// Put uploads data to S3, returns size written
func (s *S3Destination) Put(key string, reader io.Reader) (int64, error) {
	fullKey := s.getFullKey(key)

	// Use S3 manager for efficient multipart uploads
	uploader := manager.NewUploader(s.client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 10MB parts
		u.Concurrency = 3              // Limited concurrency for Raspberry Pi
	})

	// Create a custom reader that tracks bytes read
	trackingReader := &sizeTrackingReader{reader: reader}

	result, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
		Body:   trackingReader,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Log the ETag for verification
	fmt.Printf("Uploaded to S3: %s (ETag: %s)\n", fullKey, *result.ETag)

	return trackingReader.bytesRead, nil
}

// Get retrieves data from S3
func (s *S3Destination) Get(key string) (io.ReadCloser, error) {
	fullKey := s.getFullKey(key)

	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	return result.Body, nil
}

// Delete removes data from S3
func (s *S3Destination) Delete(key string) error {
	fullKey := s.getFullKey(key)

	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// List returns objects with the given prefix
func (s *S3Destination) List(prefix string) ([]btypes.BackupObject, error) {
	fullPrefix := s.getFullKey(prefix)

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fullPrefix),
	})

	var objects []btypes.BackupObject

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, btypes.BackupObject{
				Key:          s.stripPrefix(*obj.Key),
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}

	return objects, nil
}

// GetURL returns a pre-signed URL for direct access
func (s *S3Destination) GetURL(key string, expiry time.Duration) (string, error) {
	fullKey := s.getFullKey(key)

	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to create pre-signed URL: %w", err)
	}

	return request.URL, nil
}

// Type returns the destination type identifier
func (s *S3Destination) Type() string {
	return "s3"
}

// getFullKey returns the full S3 key including any prefix
func (s *S3Destination) getFullKey(key string) string {
	if s.prefix != "" {
		return s.prefix + "/" + key
	}
	return key
}

// stripPrefix removes the destination prefix from a key
func (s *S3Destination) stripPrefix(key string) string {
	if s.prefix != "" && len(key) > len(s.prefix)+1 {
		return key[len(s.prefix)+1:]
	}
	return key
}

// sizeTrackingReader tracks the number of bytes read
type sizeTrackingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *sizeTrackingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)
	return n, err
}