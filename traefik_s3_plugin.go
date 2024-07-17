package traefik_s3_plugin

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/craigbrogle/traefik-s3-plugin/local"
	"github.com/craigbrogle/traefik-s3-plugin/log"
	"github.com/craigbrogle/traefik-s3-plugin/s3"
)

type Service interface {
	Get(name string, rw http.ResponseWriter) ([]byte, error)
}

type Config struct {
	TimeoutSeconds int
	Service        string

	// Local directory
	Directory string

	// S3
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	EndpointUrl     string
	Bucket          string
	Prefix          string
}

func CreateConfig() *Config {
	return &Config{TimeoutSeconds: 5}
}

type S3Plugin struct {
	next    http.Handler
	name    string
	service Service
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	plugin := &S3Plugin{next: next, name: name}
	switch config.Service {
	case "s3":
		if config.AccessKeyId == "" {
			log.Info("AccessKeyId not configured, using AWS_ACCESS_KEY_ID")
			config.AccessKeyId = os.Getenv("AWS_ACCESS_KEY_ID")
		}
		if config.SecretAccessKey == "" {
			log.Info("SecretAccessKey not configured, using AWS_SECRET_ACCESS_KEY")
			config.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}
		if config.EndpointUrl == "" {
			log.Info("EndpointUrl not configured, using AWS_ENDPOINT_URL_S3")
			config.EndpointUrl = os.Getenv("AWS_ENDPOINT_URL_S3")
		}
		if config.Region == "" {
			log.Info("Region not configured, using AWS_DEFAULT_REGION")
			config.Region = os.Getenv("AWS_DEFAULT_REGION")
		}

		plugin.service = s3.New(config.AccessKeyId, config.SecretAccessKey, config.EndpointUrl, config.Region, config.Bucket, config.Prefix, config.TimeoutSeconds)
		return plugin, nil
	case "local":
		plugin.service = local.New(config.Directory)
		return plugin, nil
	default:
		log.Error(fmt.Sprintf("Invalid configuration: Service %s is unknown", config.Service))
	}
	return next, fmt.Errorf("invalid configuration: %v", config)
}

func (plugin S3Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		plugin.get(rw, req)
	default:
		http.Error(rw, fmt.Sprintf("Method %s not implemented", req.Method), http.StatusNotImplemented)
	}
	plugin.next.ServeHTTP(rw, req)
}

func (plugin *S3Plugin) get(rw http.ResponseWriter, req *http.Request) {
	resp, err := plugin.service.Get(req.URL.Path[1:], rw)

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		http.Error(rw, fmt.Sprintf("Put error: %s", err.Error()), http.StatusInternalServerError)
		log.Error(err.Error())
		return
	}
	rw.WriteHeader(http.StatusOK)
	_, writeError := rw.Write(resp)
	if writeError != nil {
		http.Error(rw, string(resp)+writeError.Error(), http.StatusBadGateway)
	}
}
