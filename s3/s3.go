package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3 struct {
	client         *s3.Client
	presign        *s3.PresignClient
	bucket         string
	prefix         string
	endpoint       string
	region         string
	timeoutSeconds int
}

func New(endpoint, region, bucket, prefix string, timeoutSeconds int) (*S3, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	presign := s3.NewPresignClient(client)

	return &S3{
		client:         client,
		presign:        presign,
		bucket:         bucket,
		prefix:         prefix,
		endpoint:       endpoint,
		region:         region,
		timeoutSeconds: timeoutSeconds,
	}, nil
}

// Get fetches the object from S3 and writes it to the response writer.
func (s *S3) Get(path string, rw http.ResponseWriter) ([]byte, error) {
	key := s.prefix + path
	req, err := s.presign.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		http.Error(rw, fmt.Sprintf("unable to generate presigned URL, %v", err), http.StatusInternalServerError)
		return nil, err
	}

	resp, err := http.Get(req.URL)
	if err != nil {
		http.Error(rw, fmt.Sprintf("unable to fetch object from S3, %v", err), http.StatusInternalServerError)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(rw, fmt.Sprintf("failed to fetch object from S3, status: %s", resp.Status), resp.StatusCode)
		return nil, errors.New("failed to fetch object from S3")
	}

	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

	response, err := io.ReadAll(resp.Body)
	return response, err
}
