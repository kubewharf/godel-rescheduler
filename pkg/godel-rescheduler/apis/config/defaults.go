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

package config

import (
	"net"
	"strconv"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/util/tracing"

	"k8s.io/apimachinery/pkg/runtime"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

const (
	// NamespaceSystem is the system namespace where we place system components.
	NamespaceSystem = "godel-system"

	// DefaultReschedulerName defines the name of godel rescheduler
	DefaultReschedulerName = "godel-rescheduler"

	// DefaultRenewIntervalDuration is the default value for the renew interval duration for scheduler.
	DefaultRenewIntervalDuration = 30 * time.Second

	// rescheduler qps
	DefaultClientConnectionQPS = 10000.0
	// rescheduler burst
	DefaultClientConnectionBurst = 10000

	// default idc name for godel rescheduler
	DefaultIDC = "lq"
	// default cluster name for godel rescheduler
	DefaultCluster = "default"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_GodelReschedulerConfiguration sets additional defaults
func SetDefaults_GodelReschedulerConfiguration(obj *GodelReschedulerConfiguration) {
	if obj.Profile == nil {
		obj.Profile = &ReschedulerProfile{}
	}

	if obj.Profile.IDCName == nil {
		defaultValue := DefaultIDC
		obj.Profile.IDCName = &defaultValue
	}

	if obj.Profile.ClusterName == nil {
		defaultValue := DefaultCluster
		obj.Profile.ClusterName = &defaultValue
	}

	if obj.Profile.Tracer == nil {
		defaultValue := string(tracing.NoopConfig)
		obj.Profile.Tracer = &defaultValue
	}

	// For Healthz and Metrics bind addresses, we want to check:
	// 1. If the value is nil, default to 0.0.0.0 and default scheduler port
	// 2. If there is a value set, attempt to split it. If it's just a port (ie, ":1234"), default to 0.0.0.0 with that port
	// 3. If splitting the value fails, check if the value is even a valid IP. If so, use that with the default port.
	// Otherwise use the default bind address
	defaultBindAddress := net.JoinHostPort("0.0.0.0", strconv.Itoa(DefaultInsecureReschedulerPort))
	if len(obj.HealthzBindAddress) == 0 {
		obj.HealthzBindAddress = defaultBindAddress
	} else {
		if host, port, err := net.SplitHostPort(obj.HealthzBindAddress); err == nil {
			if len(host) == 0 {
				host = "0.0.0.0"
			}
			hostPort := net.JoinHostPort(host, port)
			obj.HealthzBindAddress = hostPort
		} else {
			// Something went wrong splitting the host/port, could just be a missing port so check if the
			// existing value is a valid IP address. If so, use that with the default scheduler port
			if host := net.ParseIP(obj.HealthzBindAddress); host != nil {
				hostPort := net.JoinHostPort(obj.HealthzBindAddress, strconv.Itoa(DefaultInsecureReschedulerPort))
				obj.HealthzBindAddress = hostPort
			} else {
				obj.HealthzBindAddress = defaultBindAddress
			}
		}
	}

	if len(obj.MetricsBindAddress) == 0 {
		obj.MetricsBindAddress = defaultBindAddress
	} else {
		if host, port, err := net.SplitHostPort(obj.MetricsBindAddress); err == nil {
			if len(host) == 0 {
				host = "0.0.0.0"
			}
			hostPort := net.JoinHostPort(host, port)
			obj.MetricsBindAddress = hostPort
		} else {
			// Something went wrong splitting the host/port, could just be a missing port so check if the
			// existing value is a valid IP address. If so, use that with the default scheduler port
			if host := net.ParseIP(obj.MetricsBindAddress); host != nil {
				hostPort := net.JoinHostPort(obj.MetricsBindAddress, strconv.Itoa(DefaultInsecureReschedulerPort))
				obj.MetricsBindAddress = hostPort
			} else {
				obj.MetricsBindAddress = defaultBindAddress
			}
		}
	}

	if len(obj.LeaderElection.ResourceLock) == 0 {
		// Use lease-based leader election to reduce cost.
		// We migrated for EndpointsLease lock in 1.17 and starting in 1.20 we
		// migrated to Lease lock.
		obj.LeaderElection.ResourceLock = "leases"
	}
	if len(obj.LeaderElection.ResourceNamespace) == 0 {
		obj.LeaderElection.ResourceNamespace = NamespaceSystem
	}
	if len(obj.LeaderElection.ResourceName) == 0 {
		obj.LeaderElection.ResourceName = DefaultReschedulerName
	}

	if len(obj.ClientConnection.ContentType) == 0 {
		obj.ClientConnection.ContentType = "application/vnd.kubernetes.protobuf"
	}
	// Scheduler has an opinion about QPS/Burst, setting specific defaults for itself, instead of generic settings.
	if obj.ClientConnection.QPS == 0.0 {
		obj.ClientConnection.QPS = DefaultClientConnectionQPS
	}
	if obj.ClientConnection.Burst == 0 {
		obj.ClientConnection.Burst = DefaultClientConnectionBurst
	}

	// Use the default LeaderElectionConfiguration options
	componentbaseconfigv1alpha1.RecommendedDefaultLeaderElectionConfiguration(&obj.LeaderElection)

	// Enable profiling by default in the scheduler
	if obj.EnableProfiling == nil {
		enableProfiling := true
		obj.EnableProfiling = &enableProfiling
	}

	// Enable contention profiling by default if profiling is enabled
	if *obj.EnableProfiling && obj.EnableContentionProfiling == nil {
		enableContentionProfiling := true
		obj.EnableContentionProfiling = &enableContentionProfiling
	}
}
