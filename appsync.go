package appsync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
)

type AppSyncClient struct {
	HttpClient  *http.Client
	Credentials *aws.Credentials
	Siner       *v4.Signer
}

const SERVICE = "appsync"

var GRAPHQL_API_ID = os.Getenv("API_HEATCTRL_GRAPHQLAPIIDOUTPUT")
var GRAPHQL_ENDPOINT = os.Getenv("API_HEATCTRL_GRAPHQLAPIENDPOINTOUTPUT")
var REGION = os.Getenv("REGION")

func NewAppSyncClient() (*AppSyncClient, error) {

	println("****")
	println(GRAPHQL_API_ID)
	println(GRAPHQL_ENDPOINT)
	println(SERVICE)
	println(REGION)
	println("****")

	credentials, err := RetrieveCredentials(context.Background())
	if err != nil {
		return nil, err
	}
	// https://christina04.hatenablog.com/entry/go-keep-alive
	return &AppSyncClient{
		&http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 90 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          10,
				MaxConnsPerHost:       100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 10 * time.Second,
		},
		credentials,
		v4.NewSigner(),
	}, nil
}

func RetrieveCredentials(ctx context.Context) (*aws.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(REGION))
	if err != nil {
		return nil, err
	}
	credentials, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}

func HashBody(req *http.Request) (string, error) {
	body, err := req.GetBody()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.ReadFrom(body)
	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:]), nil
}

func (client *AppSyncClient) SendRequest(ctx context.Context, graphQL io.Reader) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, GRAPHQL_ENDPOINT, graphQL)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	hash, err := HashBody(req)
	if err != nil {
		return nil, err
	}

	client.Siner.SignHTTP(ctx, *client.Credentials, req, hash, SERVICE, REGION, time.Now())
	if err != nil {
		return nil, err
	}

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	} else {
		return nil, fmt.Errorf("Non-OK HTTP status: %v", resp.StatusCode)
	}
}
