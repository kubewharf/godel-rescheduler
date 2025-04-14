// Copyright 2024 The Godel Rescheduler Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package requests

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	tlsConfigMap  = map[string]*tls.Config{}
	tlsMapMutex   = sync.Mutex{}
	defaultClient Client
	once          = sync.Once{}
)

const (
	MethodGet          = "GET"
	MethodPost         = "POST"
	MethodPut          = "PUT"
	MethodHead         = "HEAD"
	MethodDelete       = "DELETE"
	Empty              = ""
	DefaultTimeout     = time.Second * 3
	DefaultConnTimeout = time.Second * 1
)

func GetDefault() Client {
	once.Do(func() {
		var err error
		defaultClient, err = NewHTTPClient(DefaultTimeout, WithConnTimeout(DefaultConnTimeout))
		if err != nil {
			panic(err)
		}
	})
	return defaultClient
}

type HTTPClient struct {
	Client *http.Client
}

func (c *HTTPClient) Get(ctx context.Context, url string, params map[string]string,
	headers map[string]string, builders ...ReqBuilder) *Response {
	return c.DoRequest(ctx, MethodGet, url, params, headers, nil, builders...)
}

func (c *HTTPClient) Post(ctx context.Context, url string, params map[string]string,
	headers map[string]string, payload []byte, builders ...ReqBuilder) *Response {
	reader := bytes.NewReader(payload)
	return c.DoRequest(ctx, MethodPost, url, params, headers, reader, builders...)
}

func (c *HTTPClient) Put(ctx context.Context, url string, params map[string]string,
	headers map[string]string, payload []byte, builders ...ReqBuilder) *Response {
	reader := bytes.NewReader(payload)
	return c.DoRequest(ctx, MethodPut, url, params, headers, reader, builders...)
}

func (c *HTTPClient) Head(ctx context.Context, url string, params map[string]string,
	headers map[string]string, builders ...ReqBuilder) *Response {
	return c.DoRequest(ctx, MethodHead, url, params, headers, nil, builders...)
}

func (c *HTTPClient) Delete(ctx context.Context, url string, params map[string]string,
	headers map[string]string, builders ...ReqBuilder) *Response {
	return c.DoRequest(ctx, MethodDelete, url, params, headers, nil, builders...)
}

func (c *HTTPClient) DoRequest(ctx context.Context, method string, url string, params map[string]string,
	headers map[string]string, body io.Reader, builders ...ReqBuilder) *Response {
	req, err := http.NewRequest(method, url, body)
	response := &Response{}
	runtime.SetFinalizer(response, responseFinalizer)
	if err != nil {
		response.Err = err
		return response
	}
	if params != nil {
		q := req.URL.Query()
		for key := range params {
			q.Add(key, params[key])
		}
		req.URL.RawQuery = q.Encode()
		if ctx != nil {
			req = req.WithContext(ctx)
		}
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}
	for i := range builders {
		builders[i](req)
	}
	resp, err := c.Client.Do(req)
	response.RawResponse = resp
	if err != nil {
		response.Err = err
		return response
	}
	return response
}

func WithConnTimeout(connTimeout time.Duration) TransportOpt {
	return func(transport *http.Transport) {
		transport.DialContext = (&net.Dialer{
			Timeout:   connTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext
	}
}

func WithTLSClientConfig(tlsConfig *tls.Config) TransportOpt {
	return func(transport *http.Transport) {
		transport.TLSClientConfig = tlsConfig
	}
}

func NewHTTPClient(timeout time.Duration, opts ...TransportOpt) (Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	for i := range opts {
		opts[i](transport)
	}
	return &HTTPClient{
		Client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

func LoadOrGetTLSConfig(clientCertificate, clientKey, caCertificate string) (*tls.Config, error) {
	cacheKey := clientCertificate + clientKey + caCertificate
	tlsMapMutex.Lock()
	cachedConfig, exists := tlsConfigMap[cacheKey]
	tlsMapMutex.Unlock()
	if exists {
		return cachedConfig, nil
	}

	certificate, err := tls.LoadX509KeyPair(clientCertificate, clientKey)
	if err != nil {
		return nil, fmt.Errorf("could not read client certificate: %s", err)
	}
	certPool := x509.NewCertPool()
	caCertificateClean := filepath.Clean(caCertificate)
	ca, err := ioutil.ReadFile(caCertificateClean)
	if err != nil {
		return nil, fmt.Errorf("could not read ca certificate: %s", err)
	}

	// Append the certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("failed to append ca certs %s", caCertificate)
	}
	if err != nil {
		return nil, err
	}
	c := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	}
	tlsMapMutex.Lock()
	tlsConfigMap[cacheKey] = c
	tlsMapMutex.Unlock()
	return c, nil
}
