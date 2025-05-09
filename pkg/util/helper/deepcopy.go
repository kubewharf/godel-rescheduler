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

package helper

// DeepCopyMap performs a deep copy of the given map m.
func DeepCopyMap(m map[string]string) map[string]string {
	mapCopy := make(map[string]string)
	for k, v := range m {
		mapCopy[k] = v
	}
	return mapCopy
}
