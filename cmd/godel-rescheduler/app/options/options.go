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
	"net"
	"os"
	"strconv"
	"time"

	reschedulerappconfig "github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/config"
	reschedulerserverconfig "github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/config"
	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/util/ports"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	godelreschedulerscheme "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config/scheme"
	cmdutil "github.com/kubewharf/godel-rescheduler/pkg/util/cmd"

	godelclient "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	katalystclient "github.com/kubewharf/katalyst-api/pkg/client/clientset/versioned"
	katalystinformers "github.com/kubewharf/katalyst-api/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/util/uuid"
	apiserveroptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	cliflag "k8s.io/component-base/cli/flag"
	componentbaseconfig "k8s.io/component-base/config/v1alpha1"
	"k8s.io/klog/v2"
)

var DefaultAgentName = "godel-rescheduler"

type Options struct {
	// The default values. These are overridden if ConfigFile is set or by values in InsecureServing.
	SchedulerConfig   schedulerconfig.GodelSchedulerConfiguration
	ReschedulerConfig config.GodelReschedulerConfiguration

	SecureServing           *apiserveroptions.SecureServingOptionsWithLoopback
	CombinedInsecureServing *CombinedInsecureServingOptions
	Authentication          *apiserveroptions.DelegatingAuthenticationOptions
	Authorization           *apiserveroptions.DelegatingAuthorizationOptions

	// ConfigFile is the location of the scheduler server's configuration file.
	SchedulerConfigFile string

	// WriteConfigTo is the path where the default scheduler configuration will be written.
	WriteSchedulerConfigTo string

	// ReschedulerConfigFile is the location of the rescheduler server's configuration file
	ReschedulerConfigFile string

	// WriteReschedulerConfigTo is the path where the default rescheduler configuration will be written.
	WriteReschedulerConfigTo string

	Master string

	// name of the rescheduler instance, dispatcher will distinguish schedulers by name.
	GodelReschedulerName string

	DryRun bool
}

// NewOptions returns default scheduler app options.
func NewOptions() (*Options, error) {
	reschedulerConfig, err := newDefaultComponentConfig()
	if err != nil {
		return nil, err
	}

	hhost, hport, err := splitHostIntPort(reschedulerConfig.HealthzBindAddress)
	if err != nil {
		return nil, err
	}

	o := &Options{
		ReschedulerConfig: *reschedulerConfig,
		SecureServing:     apiserveroptions.NewSecureServingOptions().WithLoopback(),
		CombinedInsecureServing: &CombinedInsecureServingOptions{
			Healthz: (&apiserveroptions.DeprecatedInsecureServingOptions{
				BindNetwork: "tcp",
			}).WithLoopback(),
			Metrics: (&apiserveroptions.DeprecatedInsecureServingOptions{
				BindNetwork: "tcp",
			}).WithLoopback(),
			BindPort:    hport,
			BindAddress: hhost,
		},
		Authentication: apiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  apiserveroptions.NewDelegatingAuthorizationOptions(),
	}

	o.Authentication.TolerateInClusterLookupFailure = true
	o.Authentication.RemoteKubeConfigFileOptional = true
	o.Authorization.RemoteKubeConfigFileOptional = true
	o.Authorization.AlwaysAllowPaths = []string{"/healthz"}

	// Set the PairName but leave certificate directory blank to generate in-memory by default
	o.SecureServing.ServerCert.CertDirectory = ""
	o.SecureServing.ServerCert.PairName = "godel-rescheduler"
	o.SecureServing.BindPort = ports.GodelReschedulerPort

	return o, nil
}

func splitHostIntPort(s string) (string, int, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, err
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	return host, portInt, err
}

func newDefaultComponentConfig() (*config.GodelReschedulerConfiguration, error) {
	cfg := config.GodelReschedulerConfiguration{}

	godelreschedulerscheme.Scheme.Default(&cfg)
	return &cfg, nil
}

// Flags returns flags for a specific scheduler by section name
func (o *Options) Flags() (nfs cliflag.NamedFlagSets) {
	fs := nfs.FlagSet("misc")
	fs.StringVar(&o.SchedulerConfigFile, "scheduler-config", o.SchedulerConfigFile, "The path to the scheduler configuration file. Flags override values in this file.")
	fs.StringVar(&o.WriteSchedulerConfigTo, "write-scheduler-config-to", o.WriteSchedulerConfigTo, "If set, write the scheduler configuration values to this file and exit.")
	fs.StringVar(&o.ReschedulerConfigFile, "rescheduler-config", o.ReschedulerConfigFile, "The path to the rescheduler configuration file. Flags override values in this file.")
	fs.StringVar(&o.WriteReschedulerConfigTo, "write-rescheduler-config-to", o.WriteReschedulerConfigTo, "If set, write the rescheduler configuration values to this file and exit.")
	fs.StringVar(&o.Master, "master", o.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&o.ReschedulerConfig.ClientConnection.Kubeconfig, "kubeconfig", o.ReschedulerConfig.ClientConnection.Kubeconfig, "path to kubeconfig file with authorization and master location information.")
	fs.StringVar(&o.GodelReschedulerName, "godel-rescheduler-name", o.GodelReschedulerName, "rescheduler name, to register scheduler crd.")
	fs.Float32Var(&o.ReschedulerConfig.ClientConnection.QPS, "kube-api-qps", o.ReschedulerConfig.ClientConnection.QPS, "QPS to use while talking with kubernetes apiserver. This parameter is ignored if a config file is specified in --config.")
	fs.Int32Var(&o.ReschedulerConfig.ClientConnection.Burst, "kube-api-burst", o.ReschedulerConfig.ClientConnection.Burst, "burst to use while talking with kubernetes apiserver. This parameter is ignored if a config file is specified in --config.")
	fs.StringVar(o.ReschedulerConfig.Profile.IDCName, "trace-idc", *o.ReschedulerConfig.Profile.IDCName, "the idc name of deployment")
	fs.StringVar(o.ReschedulerConfig.Profile.ClusterName, "trace-cluster", *o.ReschedulerConfig.Profile.ClusterName, "the cluster name of deployment")
	fs.StringVar(o.ReschedulerConfig.Profile.Tracer, "tracer", *o.ReschedulerConfig.Profile.Tracer, "tracer to use, default is bytedtrace. Options are bytedtrace, jaeger and noop.")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "If set true, rescheduler will not kill tasks")

	o.SecureServing.AddFlags(nfs.FlagSet("secure serving"))
	o.CombinedInsecureServing.AddFlags(nfs.FlagSet("insecure serving"))
	o.Authentication.AddFlags(nfs.FlagSet("authentication"))
	o.Authorization.AddFlags(nfs.FlagSet("authorization"))

	BindFlags(&o.ReschedulerConfig.LeaderElection, nfs.FlagSet("leader election"))
	utilfeature.DefaultMutableFeatureGate.AddFlag(nfs.FlagSet("feature gate"))

	return nfs
}

// ApplyTo applies the scheduler options to the given scheduler app configuration.
func (o *Options) ApplyTo(c *reschedulerserverconfig.Config) error {
	if len(o.SchedulerConfigFile) == 0 {
		c.SchedulerConfig = o.SchedulerConfig
	} else {
		cfg, err := loadSchedulerConfigFromFile(o.SchedulerConfigFile)
		if err != nil {
			return err
		}

		// use the loaded config file only, with the exception of --address and --port. This means that
		// none of the deprecated flags in o.Deprecated are taken into consideration. This is the old
		// behavior of the flags we have to keep.
		c.SchedulerConfig = *cfg
	}

	if len(o.ReschedulerConfigFile) == 0 {
		c.ReschedulerConfig = o.ReschedulerConfig

		if err := o.CombinedInsecureServing.ApplyTo(c, &c.ReschedulerConfig); err != nil {
			return err
		}
	} else {
		cfg, err := loadReschedulerConfigFromFile(o.ReschedulerConfigFile)
		if err != nil {
			return err
		}

		// use the loaded config file only, with the exception of --address and --port. This means that
		// none of the deprecated flags in o.Deprecated are taken into consideration. This is the old
		// behavior of the flags we have to keep.
		c.ReschedulerConfig = *cfg

		if err := o.CombinedInsecureServing.ApplyToFromLoadedConfig(c, &c.ReschedulerConfig); err != nil {
			return err
		}
	}

	if err := o.SecureServing.ApplyTo(&c.SecureServing, &c.LoopbackClientConfig); err != nil {
		return err
	}
	if o.SecureServing != nil && (o.SecureServing.BindPort != 0 || o.SecureServing.Listener != nil) {
		if err := o.Authentication.ApplyTo(&c.Authentication, c.SecureServing, nil); err != nil {
			return err
		}
		if err := o.Authorization.ApplyTo(&c.Authorization); err != nil {
			return err
		}
	}

	// if tracing related specified, replace config from options
	if o.ReschedulerConfig.Profile.PSM != nil {
		c.ReschedulerConfig.Profile.PSM = o.ReschedulerConfig.Profile.PSM
	}
	if o.ReschedulerConfig.Profile.Tracer != nil {
		c.ReschedulerConfig.Profile.Tracer = o.ReschedulerConfig.Profile.Tracer
	}
	if o.ReschedulerConfig.Profile.ClusterName != nil {
		c.ReschedulerConfig.Profile.ClusterName = o.ReschedulerConfig.Profile.ClusterName
	}
	if o.ReschedulerConfig.Profile.IDCName != nil {
		c.ReschedulerConfig.Profile.IDCName = o.ReschedulerConfig.Profile.IDCName
	}

	// if client connection configuration is set, replace config from options
	if o.ReschedulerConfig.ClientConnection.QPS != 0 {
		c.ReschedulerConfig.ClientConnection.QPS = o.ReschedulerConfig.ClientConnection.QPS
	}
	if o.ReschedulerConfig.ClientConnection.Burst != 0 {
		c.ReschedulerConfig.ClientConnection.Burst = o.ReschedulerConfig.ClientConnection.Burst
	}
	if o.ReschedulerConfig.ClientConnection.Kubeconfig != "" {
		c.ReschedulerConfig.ClientConnection.Kubeconfig = o.ReschedulerConfig.ClientConnection.Kubeconfig
	}

	// if leader election configuration is set, replace config from options
	if o.ReschedulerConfig.LeaderElection.LeaderElect != nil {
		c.ReschedulerConfig.LeaderElection.LeaderElect = o.ReschedulerConfig.LeaderElection.LeaderElect
	}
	if o.ReschedulerConfig.LeaderElection.ResourceLock != "" {
		c.ReschedulerConfig.LeaderElection.ResourceLock = o.ReschedulerConfig.LeaderElection.ResourceLock
	}
	if o.ReschedulerConfig.LeaderElection.ResourceNamespace != "" {
		c.ReschedulerConfig.LeaderElection.ResourceNamespace = o.ReschedulerConfig.LeaderElection.ResourceNamespace
	}
	if o.ReschedulerConfig.LeaderElection.ResourceName != "" {
		c.ReschedulerConfig.LeaderElection.ResourceName = o.ReschedulerConfig.LeaderElection.ResourceName
	}

	// check listen port and override is not default
	if o.CombinedInsecureServing.BindPort != 0 || o.CombinedInsecureServing.BindAddress != "" {
		if err := o.CombinedInsecureServing.ApplyTo(c, &c.ReschedulerConfig); err != nil {
			return err
		}
	} else if err := o.CombinedInsecureServing.ApplyToFromLoadedConfig(c, &c.ReschedulerConfig); err != nil {
		return err
	}

	return nil
}

// Validate validates all the required options.
func (o *Options) Validate() []error {
	var errs []error
	errs = append(errs, o.SecureServing.Validate()...)
	errs = append(errs, o.CombinedInsecureServing.Validate()...)
	errs = append(errs, o.Authentication.Validate()...)
	errs = append(errs, o.Authorization.Validate()...)
	return errs
}

// Config return a scheduler config object
func (o *Options) Config() (*reschedulerappconfig.Config, error) {
	if o.SecureServing != nil {
		if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
			return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
		}
	}

	c := &reschedulerappconfig.Config{}
	if err := o.ApplyTo(c); err != nil {
		return nil, err
	}

	client, leaderElectionClient, eventClient, godelCrdClient, katalystCrdClient, err := createClients(c.ReschedulerConfig.ClientConnection, o.Master, c.ReschedulerConfig.LeaderElection.RenewDeadline.Duration)
	if err != nil {
		return nil, err
	}

	c.EventBroadcaster = cmdutil.NewEventBroadcasterAdapter(eventClient)

	// Set up leader election if enabled.
	var leaderElectionConfig *leaderelection.LeaderElectionConfig
	if *c.ReschedulerConfig.LeaderElection.LeaderElect {
		// Use the scheduler name in the first profile to record leader election.
		coreRecorder := c.EventBroadcaster.DeprecatedNewLegacyRecorder(config.DefaultReschedulerName)
		leaderElectionConfig, err = makeLeaderElectionConfig(c.ReschedulerConfig.LeaderElection, leaderElectionClient, coreRecorder)
		if err != nil {
			return nil, err
		}
	}

	c.Client = client
	c.InformerFactory = cmdutil.NewInformerFactory(client, 0)
	c.GodelCrdClient = godelCrdClient
	c.GodelCrdInformerFactory = crdinformers.NewSharedInformerFactory(c.GodelCrdClient, 0)
	c.KatalystCrdInformerFactory = katalystinformers.NewSharedInformerFactory(katalystCrdClient, 0)

	c.LeaderElection = leaderElectionConfig
	c.DryRun = o.DryRun

	return c, nil
}

// makeLeaderElectionConfig builds a leader election configuration. It will
// create a new resource lock associated with the configuration.
func makeLeaderElectionConfig(config componentbaseconfig.LeaderElectionConfiguration, client clientset.Interface, recorder record.EventRecorder) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New(config.ResourceLock,
		config.ResourceNamespace,
		config.ResourceName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		return nil, fmt.Errorf("couldn't create resource lock: %v", err)
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: config.LeaseDuration.Duration,
		RenewDeadline: config.RenewDeadline.Duration,
		RetryPeriod:   config.RetryPeriod.Duration,
		WatchDog:      leaderelection.NewLeaderHealthzAdaptor(time.Second * 20),
		Name:          fmt.Sprintf("%s/%s", config.ResourceNamespace, config.ResourceName),
	}, nil
}

// createClients creates a kube client and an event client from the given config and masterOverride.
// TODO remove masterOverride when CLI flags are removed.
func createClients(config componentbaseconfig.ClientConnectionConfiguration, masterOverride string, timeout time.Duration) (clientset.Interface, clientset.Interface, clientset.Interface, godelclient.Interface, katalystclient.Interface, error) {
	if len(config.Kubeconfig) == 0 && len(masterOverride) == 0 {
		klog.Warningf("Neither --kubeconfig nor --master was specified. Using default API client. This might not work.")
	}

	// This creates a client, if kubeconfig file and master are not configured, we fallback to inClusterConfig
	// otherwise, first loading any specified kubeconfig
	// file, and then overriding the Master flag, if non-empty.
	kubeConfig, err := clientcmd.BuildConfigFromFlags(masterOverride, config.Kubeconfig)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	kubeConfig.DisableCompression = true
	kubeConfig.AcceptContentTypes = config.AcceptContentTypes
	kubeConfig.ContentType = config.ContentType
	kubeConfig.QPS = config.QPS
	// TODO make config struct use int instead of int32?
	kubeConfig.Burst = int(config.Burst)

	client, err := clientset.NewForConfig(restclient.AddUserAgent(kubeConfig, DefaultAgentName))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// shallow copy, do not modify the kubeConfig.Timeout.
	restConfig := *kubeConfig
	restConfig.Timeout = timeout
	leaderElectionClient, err := clientset.NewForConfig(restclient.AddUserAgent(&restConfig, "leader-election"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	eventClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	//utilruntime.Must(godelclientscheme.AddToScheme(clientsetscheme.Scheme))
	// This creates a client, first loading any specified kubeconfig
	// file, and then overriding the Master flag, if non-empty.
	crdKubeConfig, err := clientcmd.BuildConfigFromFlags(masterOverride, config.Kubeconfig)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	crdKubeConfig.DisableCompression = true
	crdKubeConfig.QPS = config.QPS
	// TODO make config struct use int instead of int32?
	crdKubeConfig.Burst = int(config.Burst)

	godelCrdClient, err := godelclient.NewForConfig(restclient.AddUserAgent(crdKubeConfig, DefaultAgentName))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	katalystCrdClient, _ := katalystclient.NewForConfig(restclient.AddUserAgent(crdKubeConfig, DefaultAgentName))
	return client, leaderElectionClient, eventClient, godelCrdClient, katalystCrdClient, nil
}
