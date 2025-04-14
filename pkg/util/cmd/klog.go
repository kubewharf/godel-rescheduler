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

package cmd

import (
	"flag"

	"github.com/spf13/pflag"
	klogv1 "k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
)

// OutputCallDepth is the stack depth where we can find the origin of this call
const OutputCallDepth = 6

// DefaultPrefixLength is the length of the log prefix that we have to strip out
const DefaultPrefixLength = 42

// klogWriter is used in SetOutputBySeverity call below to redirect
// any calls to klogv1 to end up in klogv2
type klogWriter struct{}

func (kw klogWriter) Write(p []byte) (n int, err error) {
	if len(p) < DefaultPrefixLength {
		klogv1.InfoDepth(OutputCallDepth, string(p))
		return len(p), nil
	}
	switch p[0] {
	case 'I':
		klogv1.InfoDepth(OutputCallDepth, string(p[DefaultPrefixLength:]))
	case 'W':
		klogv1.WarningDepth(OutputCallDepth, string(p[DefaultPrefixLength:]))
	case 'E':
		klogv1.ErrorDepth(OutputCallDepth, string(p[DefaultPrefixLength:]))
	case 'F':
		klogv1.FatalDepth(OutputCallDepth, string(p[DefaultPrefixLength:]))
	default:
		klogv1.InfoDepth(OutputCallDepth, string(p[DefaultPrefixLength:]))
	}
	return len(p), nil
}

func InitKlogV2WithV1Flags(cmdFlags *pflag.FlagSet) {
	var klogv2Flags flag.FlagSet
	klogv2.InitFlags(&klogv2Flags)
	cmdFlags.VisitAll(func(f *pflag.Flag) {
		klogv2Flags.Set(f.Name, f.Value.String())
	})

	klogv2.SetOutputBySeverity("INFO", klogWriter{})
	return
}
