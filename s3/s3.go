package s3

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Helper function to create the HMAC-SHA256 hash
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// Helper function to create the hex-encoded HMAC-SHA256 hash
func hmacSHA256Hex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

// Hash function for the payload
func hashSHA256(payload string) []byte {
	h := sha256.New()
	h.Write([]byte(payload))
	return h.Sum(nil)
}

// Generate a presigned URL for an S3 GetObject request
func generatePresignedURL(accessKeyID, secretAccessKey, region, endpoint, bucket, key string, duration time.Duration) (string, error) {
	urlStr := fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, key)
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	t := time.Now().UTC()
	date := t.Format("20060102")
	timestamp := t.Format("20060102T150405Z")

	// Query parameters for the presigned URL
	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request", accessKeyID, date, region))
	query.Set("X-Amz-Date", timestamp)
	// query.Set("X-Amz-Expires", fmt.Sprintf("%d", int(duration.Seconds())))
	query.Set("X-Amz-Expires", "86400")
	query.Set("X-Amz-SignedHeaders", "host")

	// Canonical request components
	canonicalURI := parsedURL.Path
	canonicalQueryString := query.Encode()
	canonicalHeaders := fmt.Sprintf("host:%s\n", parsedURL.Host)
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"

	canonicalRequest := strings.Join([]string{
		"GET",
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// Create the string to sign
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		timestamp,
		credentialScope,
		hex.EncodeToString(hashSHA256(canonicalRequest)),
	}, "\n")

	// Calculate the signature
	signingKey := hmacSHA256([]byte("AWS4"+secretAccessKey), date)
	signingKey = hmacSHA256(signingKey, region)
	signingKey = hmacSHA256(signingKey, "s3")
	signingKey = hmacSHA256(signingKey, "aws4_request")
	signature := hmacSHA256Hex(signingKey, stringToSign)

	// Add the signature to the query parameters
	query.Set("X-Amz-Signature", signature)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

type S3 struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	endpoint        string
	bucket          string
	prefix          string
	timeoutSeconds  int
}

func New(accessKeyID, secretAccessKey, endpoint, region, bucket, prefix string, timeoutSeconds int) *S3 {
	return &S3{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		region:          region,
		endpoint:        endpoint,
		bucket:          bucket,
		prefix:          prefix,
		timeoutSeconds:  timeoutSeconds,
	}
}

// Get fetches the object from S3 and writes it to the response writer.
func (s *S3) Get(path string, rw http.ResponseWriter) ([]byte, error) {
	key := s.prefix + path
	duration := 15 * time.Minute

	urlStr, err := generatePresignedURL(s.accessKeyID, s.secretAccessKey, s.region, s.endpoint, s.bucket, key, duration)
	if err != nil {
		http.Error(rw, fmt.Sprintf("unable to generate presigned URL, %v", err), http.StatusInternalServerError)
		return nil, err
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		http.Error(rw, fmt.Sprintf("unable to fetch object from S3, %v", err), http.StatusInternalServerError)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(rw, string(body), resp.StatusCode)
		return nil, errors.New("failed to fetch object from S3")
	}

	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

	response, err := io.ReadAll(resp.Body)
	return response, err
}
