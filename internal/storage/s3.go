// Package storage handles file uploads to S3-compatible object storage.
// Configured for Hetzner Object Storage but works with any S3-compatible provider.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client uploads files to an S3-compatible bucket.
type Client struct {
	s3     *s3.Client
	bucket string
	// publicBase is prepended to object keys to form the public URL.
	// For Hetzner: https://<bucket>.<endpoint>
	publicBase string
}

// NewClient creates an S3 Client pointed at a custom endpoint.
func NewClient(endpoint, accessKey, secretKey, bucket, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://" + endpoint)
		o.UsePathStyle = false
	})

	// Hetzner public URL pattern: https://<bucket>.<endpoint>/<key>
	publicBase := fmt.Sprintf("https://%s.%s", bucket, endpoint)

	return &Client{s3: client, bucket: bucket, publicBase: publicBase}, nil
}

// Upload stores data under a generated key and returns the public URL.
// ext should include the dot, e.g. ".jpg".
func (c *Client) Upload(ctx context.Context, data []byte, ext string) (string, error) {
	key := fmt.Sprintf("news/%d%s", time.Now().UnixMilli(), ext)

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         "public-read",
	})
	if err != nil {
		return "", fmt.Errorf("upload to s3: %w", err)
	}

	return c.publicBase + "/" + key, nil
}
