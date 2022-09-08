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

func NewAppSyncClient() *AppSyncClient {
	println(GRAPHQL_API_ID)
	println(GRAPHQL_ENDPOINT)
	println(REGION)
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
		RetrieveCredentials(context.Background()),
		v4.NewSigner(),
	}
}

func RetrieveCredentials(ctx context.Context) *aws.Credentials {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(REGION))
	if err != nil {
		panic(err)
	}
	credentials, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		panic(err)
	}
	return &credentials
}

func HashBody(req *http.Request) string {
	body, err := req.GetBody()
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(body)
	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:])
}

func (client *AppSyncClient) SendRequest(ctx context.Context, graphQL io.Reader) []byte {
	req, err := http.NewRequest(http.MethodPost, GRAPHQL_ENDPOINT, graphQL)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client.Siner.SignHTTP(ctx, *client.Credentials, req, HashBody(req), SERVICE, REGION, time.Now())
	if err != nil {
		panic(err)
	}

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode == http.StatusOK {
		return respBody
	} else {
		panic(fmt.Errorf("Non-OK HTTP status: %v", resp.StatusCode))
	}
}
