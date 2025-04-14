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
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	componentbaseconfig "k8s.io/component-base/config/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	// ReschedulerDefaultLockObjectNamespace defines rescheduler lock object namespace ("godel-system")
	ReschedulerDefaultLockObjectNamespace = "godel-system"

	// ReschedulerDefaultLockObjectName defines rescheduler lock object name ("godel-rescheduler")
	ReschedulerDefaultLockObjectName = "godel-rescheduler"

	// ReschedulerPolicyConfigMapKey defines the key of the element in the
	// rescheduler's policy ConfigMap that contains rescheduler's policy config.
	ReschedulerPolicyConfigMapKey = "policy.cfg"

	// DefaultInsecureReschedulerPort is the default port for the rescheduler status server.
	// May be overridden by a flag at startup.
	// Deprecated: use the secure GodelReschedulerPort instead.
	DefaultInsecureReschedulerPort = 11251

	// GodelReschedulerPort is the default port for the rescheduler status server.
	// May be overridden by a flag at startup.
	GodelReschedulerPort = 11259
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GodelReschedulerConfiguration configures a rescheduler
type GodelReschedulerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// LeaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration `json:"leaderElection"`

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection componentbaseconfig.ClientConnectionConfiguration `json:"clientConnection"`
	// HealthzBindAddress is the IP address and port for the health check server to serve on,
	// defaulting to 0.0.0.0:10251
	HealthzBindAddress string `json:"healthzBindAddress,omitempty"`
	// MetricsBindAddress is the IP address and port for the metrics server to
	// serve on, defaulting to 0.0.0.0:10251.
	MetricsBindAddress string `json:"metricsBindAddress,omitempty"`

	// DebuggingConfiguration holds configuration for Debugging related features
	componentbaseconfig.DebuggingConfiguration `json:",inline"`

	// Profiles are scheduling profiles that godel-rescheduler supports.
	Profile *ReschedulerProfile `json:"profile"`
}

// ReschedulerProfile is a scheduling profile.
type ReschedulerProfile struct {
	// PSM specifies the name of rescheduler
	Name string `json:"name,omitempty"`

	// PSM specifies the index name for tracing
	PSM *string `json:"psm,omitempty"`

	// IDCName specifies the name of idc to deploy
	IDCName *string `json:"idcName,omitempty"`

	// ClusterName speccifies the name of cluster to serve
	ClusterName *string `json:"clusterName,omitempty"`

	// Tracer defines to enable tracing or not
	Tracer *string `json:"tracer,omitempty"`

	// Rescheduler policy configs
	ReschedulerPolicyConfigs []ReschedulerPolicy `json:"reschedulerPolicyConfigs"`
}

type ReschedulerPolicy struct {
	// trigger policy
	PolicyTrigger *PolicyTrigger `json:"policyTrigger"`
	// detect nodes and check whether need to reschedule pods & which pods need to be scheduled
	Detector DetectorConfig `json:"detector"`
	// reschedule pods
	Algorithm AlgorithmConfig `json:"algorithm"`
	// node scope, used to select nodes, if it is nil, scope is all nodes in cluster
	// +optional
	Scope map[string]string `json:"scope,omitempty"`
	// if generated movements number is larger than this value, skip them
	// +optional
	FailureRatioThreshold *float64 `json:"failureRatioThreshold,omitempty"`
}

type DetectorConfig struct {
	// name of detector plugin
	Name string `json:"name"`
	// args for detector plugin
	// +optional
	Args runtime.RawExtension `json:"args,omitempty"`
	// trigger detect plugin
	DetectorTrigger *PolicyTrigger `json:"detectorTrigger"`
	// check policies
	// +optional
	CheckPolicies []string `json:"checkPolicies,omitempty"`
}

type PolicyTrigger struct {
	// cronjob triggers
	// +optional
	CronjobTriggers []CronjobTrigger `json:"cronjobTriggers,omitempty"`
	// port for http call triggers
	// +optional
	Port *int `json:"port,omitempty"`
	// period for periodical trigger
	// +optional
	Period *string `json:"period,omitempty"`
	// signal for signal trigger
	// +optional
	Signal *int `json:"signal,omitempty"`
}

type CronjobTrigger struct {
	// crobjob time
	Time string `json:"time"`
}

type AlgorithmConfig struct {
	// name of algorithm plugin
	Name string `json:"name"`
	// args for algorithm plugin
	// +optional
	Args runtime.RawExtension `json:"args,omitempty"`
	// check policies
	// +optional
	CheckPolicies []string `json:"checkPolicies,omitempty"`
}

// DecodeNestedObjects decodes plugin args for known types.
func (in *GodelReschedulerConfiguration) DecodeNestedObjects(d runtime.Decoder) error {
	prof := in.Profile
	for j := range prof.ReschedulerPolicyConfigs {
		err := prof.ReschedulerPolicyConfigs[j].Detector.decodeNestedObjects(d)
		if err != nil {
			return fmt.Errorf("decoding .profile.reschedulerPolicyConfigs[%d].Detector: %w", j, err)
		}
		err = prof.ReschedulerPolicyConfigs[j].Algorithm.decodeNestedObjects(d)
		if err != nil {
			return fmt.Errorf("decoding .profile.reschedulerPolicyConfigs[%d].Algorithm: %w", j, err)
		}
	}
	return nil
}

// EncodeNestedObjects encodes plugin args.
func (in *GodelReschedulerConfiguration) EncodeNestedObjects(e runtime.Encoder) error {
	prof := in.Profile
	for j := range prof.ReschedulerPolicyConfigs {
		err := prof.ReschedulerPolicyConfigs[j].Detector.encodeNestedObjects(e)
		if err != nil {
			return fmt.Errorf("encoding .profile.reschedulerPolicyConfigs[%d]: %w", j, err)
		}
		err = prof.ReschedulerPolicyConfigs[j].Algorithm.encodeNestedObjects(e)
		if err != nil {
			return fmt.Errorf("decoding .profile.reschedulerPolicyConfigs[%d].Algorithm: %w", j, err)
		}
	}
	return nil
}

func (c *DetectorConfig) decodeNestedObjects(d runtime.Decoder) error {
	gvk := SchemeGroupVersion.WithKind(c.Name + "Args")
	// dry-run to detect and skip out-of-tree plugin args.
	if _, _, err := d.Decode(nil, &gvk, nil); runtime.IsNotRegisteredError(err) {
		return nil
	}

	obj, parsedGvk, err := d.Decode(c.Args.Raw, &gvk, nil)
	if err != nil {
		return fmt.Errorf("decoding args for plugin %s: %w", c.Name, err)
	}
	if parsedGvk.GroupKind() != gvk.GroupKind() {
		return fmt.Errorf("args for plugin %s were not of type %s, got %s", c.Name, gvk.GroupKind(), parsedGvk.GroupKind())
	}
	c.Args.Object = obj
	return nil
}

func (c *DetectorConfig) encodeNestedObjects(e runtime.Encoder) error {
	if c.Args.Object == nil {
		return nil
	}
	var buf bytes.Buffer
	err := e.Encode(c.Args.Object, &buf)
	if err != nil {
		return err
	}
	// The <e> encoder might be a YAML encoder, but the parent encoder expects
	// JSON output, so we convert YAML back to JSON.
	// This is a no-op if <e> produces JSON.
	json, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return err
	}
	c.Args.Raw = json
	return nil
}

func (c *AlgorithmConfig) decodeNestedObjects(d runtime.Decoder) error {
	gvk := SchemeGroupVersion.WithKind(c.Name + "Args")
	// dry-run to detect and skip out-of-tree plugin args.
	if _, _, err := d.Decode(nil, &gvk, nil); runtime.IsNotRegisteredError(err) {
		return nil
	}

	obj, parsedGvk, err := d.Decode(c.Args.Raw, &gvk, nil)
	if err != nil {
		return fmt.Errorf("decoding args for plugin %s: %w", c.Name, err)
	}
	if parsedGvk.GroupKind() != gvk.GroupKind() {
		return fmt.Errorf("args for plugin %s were not of type %s, got %s", c.Name, gvk.GroupKind(), parsedGvk.GroupKind())
	}
	c.Args.Object = obj
	return nil
}

func (c *AlgorithmConfig) encodeNestedObjects(e runtime.Encoder) error {
	if c.Args.Object == nil {
		return nil
	}
	var buf bytes.Buffer
	err := e.Encode(c.Args.Object, &buf)
	if err != nil {
		return err
	}
	// The <e> encoder might be a YAML encoder, but the parent encoder expects
	// JSON output, so we convert YAML back to JSON.
	// This is a no-op if <e> produces JSON.
	json, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return err
	}
	c.Args.Raw = json
	return nil
}
