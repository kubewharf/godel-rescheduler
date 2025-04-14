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

package options

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	godelreschedulerscheme "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config/scheme"

	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerscheme "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config/scheme"
	v1beta1 "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/component-base/codec"
	"k8s.io/klog"
)

func loadSchedulerConfigFromFile(file string) (*schedulerconfig.GodelSchedulerConfiguration, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return loadSchedulerConfig(data)
}

func loadSchedulerConfig(data []byte) (*schedulerconfig.GodelSchedulerConfiguration, error) {
	// The UniversalDecoder runs defaulting and returns the internal type by default.
	obj, gvk, err := schedulerscheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	if cfgObj, ok := obj.(*schedulerconfig.GodelSchedulerConfiguration); ok {
		cfgObj.TypeMeta.APIVersion = gvk.GroupVersion().String()
		switch cfgObj.TypeMeta.APIVersion {
		case v1beta1.SchemeGroupVersion.String():
			fmt.Printf("GodelSchedulerConfiguration v1beta1 is loaded.\n")
		}

		return cfgObj, nil
	}
	return nil, fmt.Errorf("couldn't decode as GodelSchedulerConfiguration, got %s: ", gvk)
}

func loadReschedulerConfigFromFile(file string) (*config.GodelReschedulerConfiguration, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return loadReschedulerConfig(data)
}

func loadReschedulerConfig(data []byte) (*config.GodelReschedulerConfiguration, error) {
	// The UniversalDecoder runs defaulting and returns the internal type by default.
	obj, gvk, err := godelreschedulerscheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		// Try strict decoding first. If that fails decode with a lenient
		// decoder, which has only v1alpha1 registered, and log a warning.
		// The lenient path is to be dropped when support for v1alpha1 is dropped.
		if !runtime.IsStrictDecodingError(err) {
			return nil, err
		}

		var lenientErr error
		_, lenientCodecs, lenientErr := codec.NewLenientSchemeAndCodecs(
			config.AddToScheme,
		)
		if lenientErr != nil {
			return nil, lenientErr
		}
		obj, gvk, lenientErr = lenientCodecs.UniversalDecoder().Decode(data, nil, nil)
		if lenientErr != nil {
			return nil, err
		}
		klog.Warningf("using lenient decoding as strict decoding failed: %v", err)
	}
	if cfgObj, ok := obj.(*config.GodelReschedulerConfiguration); ok {
		return cfgObj, nil
	}
	return nil, fmt.Errorf("couldn't decode as GodelReschedulerConfiguration, got %s: ", gvk)
}

// WriteSchedulerConfigFile writes the scheduler config into the given file name as YAML.
func WriteSchedulerConfigFile(fileName string, cfg *schedulerconfig.GodelSchedulerConfiguration) error {
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(schedulerscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return fmt.Errorf("unable to locate encoder -- %q is not a supported media type", mediaType)
	}

	encoder := schedulerscheme.Codecs.EncoderForVersion(info.Serializer, schedulerconfig.SchemeGroupVersion)

	configFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err := encoder.Encode(cfg, configFile); err != nil {
		return err
	}

	return nil
}

// WriteReschedulerConfigFile writes the rescheduler config into the given file name as YAML.
func WriteReschedulerConfigFile(fileName string, cfg *config.GodelReschedulerConfiguration) error {
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(godelreschedulerscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return fmt.Errorf("unable to locate encoder -- %q is not a supported media type", mediaType)
	}

	encoder := godelreschedulerscheme.Codecs.EncoderForVersion(info.Serializer, config.SchemeGroupVersion)

	configFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err := encoder.Encode(cfg, configFile); err != nil {
		return err
	}

	return nil
}
