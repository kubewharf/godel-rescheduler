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

package godel_rescheduler

import (
	reschedulerconfig "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
)

type reschedulerOptions struct {
	policyConfigs []reschedulerconfig.ReschedulerPolicy
	dryRun        bool
}

// Option configures a Scheduler
type Option func(*reschedulerOptions)

var defaultReschedulerOptions = reschedulerOptions{}

func renderOptions(opts ...Option) reschedulerOptions {
	options := defaultReschedulerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// WithReschedulerPolicyConfigs set rescheduler policy configs
func WithReschedulerPolicyConfigs(policyConfigs []reschedulerconfig.ReschedulerPolicy) Option {
	return func(o *reschedulerOptions) {
		o.policyConfigs = policyConfigs
	}
}

func WithDryRun(dryRun bool) Option {
	return func(o *reschedulerOptions) {
		o.dryRun = dryRun
	}
}
