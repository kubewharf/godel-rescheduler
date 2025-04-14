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
	"strings"

	"github.com/pkg/errors"
)

const (
	connectionRefused = "connect: connection refused"
)

func IsConnectionRefused(response *Response) bool {
	if response.Err == nil {
		return false
	}
	return strings.Contains(response.Err.Error(), connectionRefused)
}

func ParseResponseError(response *Response) error {
	if response == nil {
		return errors.Errorf("response is nil")
	} else if response.Err != nil {
		return errors.Errorf("response error: %s", response.Err)
	} else if response.RawResponse == nil {
		return errors.Errorf("raw response is nil")
	}

	return nil
}
