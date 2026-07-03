package uploader

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

type s3Uploader struct {
	cfg     config.S3UploadCfg
	timeout time.Duration
}

func newS3Uploader(cfg config.S3UploadCfg, timeout time.Duration) Uploader {
	return &s3Uploader{cfg: cfg, timeout: uploadTimeout(timeout)}
}

func (u *s3Uploader) Provider() string { return "s3" }

func (u *s3Uploader) Upload(ctx context.Context, archivePath string, summary collector.Summary) (*Result, error) {
	bucket := strings.TrimSpace(u.cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	key := objectKey(u.cfg.KeyPrefix, archivePath)
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	client, err := u.s3Client(ctx)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String(firstNonEmpty(strings.TrimSpace(u.cfg.ContentType), "application/gzip")),
	}
	if sc := strings.TrimSpace(u.cfg.StorageClass); sc != "" {
		input.StorageClass = types.StorageClass(sc)
	}
	if sse := strings.TrimSpace(u.cfg.SSE); sse != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(sse)
	}
	if kid := strings.TrimSpace(u.cfg.SSEKMSKeyID); kid != "" {
		input.SSEKMSKeyId = aws.String(kid)
	}
	if acl := strings.TrimSpace(u.cfg.ACL); acl != "" {
		input.ACL = types.ObjectCannedACL(acl)
	}
	input.Metadata = mergeMetadata(u.cfg.Metadata, summary.RunID)

	up := manager.NewUploader(client)
	out, err := up.Upload(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}

	etag := ""
	if out.ETag != nil {
		etag = *out.ETag
	}
	uri := fmt.Sprintf("s3://%s/%s", bucket, key)
	return &Result{Provider: "s3", URI: uri, Key: key, ETag: etag, Size: info.Size()}, nil
}

func (u *s3Uploader) s3Client(ctx context.Context) (*s3.Client, error) {
	region := firstNonEmpty(strings.TrimSpace(u.cfg.Region), os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), "us-east-1")
	opts := []func(*awscfg.LoadOptions) error{
		awscfg.WithRegion(region),
	}
	if id, secret := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"); id != "" && secret != "" {
		opts = append(opts, awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(id, secret, os.Getenv("AWS_SESSION_TOKEN")),
		))
	}
	loaded, err := awscfg.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return s3.NewFromConfig(loaded, func(o *s3.Options) {
		if ep := strings.TrimSpace(u.cfg.Endpoint); ep != "" {
			o.BaseEndpoint = aws.String(ep)
			o.UsePathStyle = true
		}
	}), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// mergeMetadata combines user-configured metadata with the run_id from #81.
// User-supplied keys take priority (can override run_id if desired).
func mergeMetadata(base map[string]string, runID string) map[string]string {
	out := make(map[string]string, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	if runID != "" && runID != "unknown" {
		if _, ok := out["run_id"]; !ok {
			out["run_id"] = runID
		}
	}
	return out
}
