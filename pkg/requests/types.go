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
	"context"
	"io"
	"net/http"
)

type Client interface {
	Get(ctx context.Context, url string, params map[string]string,
		headers map[string]string, builders ...ReqBuilder) *Response
	Post(ctx context.Context, url string, params map[string]string,
		headers map[string]string, payload []byte, builders ...ReqBuilder) *Response
	Put(ctx context.Context, url string, params map[string]string,
		headers map[string]string, payload []byte, builders ...ReqBuilder) *Response
	Head(ctx context.Context, url string, params map[string]string,
		headers map[string]string, builders ...ReqBuilder) *Response
	Delete(ctx context.Context, url string, params map[string]string,
		headers map[string]string, builders ...ReqBuilder) *Response

	// Do Network request
	DoRequest(ctx context.Context, method string, url string, params map[string]string, headers map[string]string, body io.Reader, builders ...ReqBuilder) *Response
}

type Response struct {
	RawResponse *http.Response
	Err         error
}

type TransportOpt func(transport *http.Transport)

type ReqBuilder func(r *http.Request)
