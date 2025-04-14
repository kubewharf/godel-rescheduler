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
	"encoding/json"
	"io/ioutil"
)

func responseFinalizer(r *Response) {
	if r.RawResponse != nil && r.RawResponse.Body != nil {
		_ = r.RawResponse.Body.Close()
	}
}

func (r *Response) JSON(v interface{}) error {
	data, err := r.Content()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &v)
}

func (r *Response) Content() ([]byte, error) {
	data, err := ioutil.ReadAll(r.RawResponse.Body)
	_ = r.RawResponse.Body.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Response) Text() (string, error) {
	data, err := ioutil.ReadAll(r.RawResponse.Body)
	_ = r.RawResponse.Body.Close()
	if err != nil {
		return Empty, err
	}
	return string(data), nil
}
