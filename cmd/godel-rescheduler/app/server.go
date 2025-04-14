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

package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	goruntime "runtime"

	reschedulerserverconfig "github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/config"
	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/options"
	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/util/configz"
	godelrescheduler "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler"
	godelreschedulerconfig "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	cmdutil "github.com/kubewharf/godel-rescheduler/pkg/util/cmd"
	"github.com/kubewharf/godel-rescheduler/pkg/util/tracing"
	"github.com/kubewharf/godel-rescheduler/pkg/version/verflag"

	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/tools/leaderelection"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
)

const ComponentName = "godel-rescheduler"

func NewGodelReschedulerCmd() *cobra.Command {
	opts, err := options.NewOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to initialize command options: %v\n", err)
		os.Exit(1)
	}

	godelreschedulerCmd := &cobra.Command{
		Use: ComponentName,
		Long: `After long-term running, state of the cluster may not fit for pods. Primary goal for godel 
rescheduler is to harvest the underutilized resources in ongoing cluster`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCommand(cmd, opts, args); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	fs := godelreschedulerCmd.Flags()
	namedFlagSets := opts.Flags()
	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), godelreschedulerCmd.Name())
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(godelreschedulerCmd.OutOrStdout())
	godelreschedulerCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	godelreschedulerCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
	godelreschedulerCmd.MarkFlagFilename("config", "yaml", "yml", "json")

	return godelreschedulerCmd
}

func runCommand(cmd *cobra.Command, opts *options.Options, args []string) error {
	cmdutil.InitKlogV2WithV1Flags(cmd.Flags())
	verflag.PrintAndExitIfRequested()
	if len(args) != 0 {
		fmt.Fprint(os.Stderr, "arguments are not supported\n")
	}

	if errs := opts.Validate(); len(errs) > 0 {
		return utilerrors.NewAggregate(errs)
	}

	c, err := opts.Config()
	if err != nil {
		return err
	}

	if len(opts.WriteSchedulerConfigTo) > 0 {
		c := &reschedulerserverconfig.Config{}
		if err := opts.ApplyTo(c); err != nil {
			return err
		}
		if err := options.WriteSchedulerConfigFile(opts.WriteSchedulerConfigTo, &c.SchedulerConfig); err != nil {
			return err
		}
		klog.Infof("Wrote scheduler configuration to: %s\n", opts.WriteSchedulerConfigTo)
	}

	if len(opts.WriteReschedulerConfigTo) > 0 {
		c := &reschedulerserverconfig.Config{}
		if err := opts.ApplyTo(c); err != nil {
			return err
		}
		if err := options.WriteReschedulerConfigFile(opts.WriteReschedulerConfigTo, &c.ReschedulerConfig); err != nil {
			return err
		}
		klog.Infof("Wrote rescheduler configuration to: %s\n", opts.WriteReschedulerConfigTo)
		return nil
	}

	// Get the completed config
	cc := c.Complete()

	// Configz registration.
	if cz, err := configz.New("componentconfig"); err == nil {
		cz.Set(cc.ReschedulerConfig)
	} else {
		return fmt.Errorf("unable to register configz: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return Run(ctx, cc)
}

func Run(ctx context.Context, cc reschedulerserverconfig.CompletedConfig) error {
	tracer, err := tracing.ValidateTracerConfig(tracing.TracerConfig(*cc.ReschedulerConfig.Profile.Tracer))
	if err != nil {
		return err
	}

	closer := tracing.NewTracer(
		tracer,
		ComponentName,
		*cc.ReschedulerConfig.Profile.IDCName,
		*cc.ReschedulerConfig.Profile.ClusterName)
	defer closer.Close()

	eventRecorder := getEventRecorder(&cc)

	// Create the rescheduler.
	rescheduler, err := godelrescheduler.New(
		cc.Client,
		cc.GodelCrdClient,
		cc.InformerFactory,
		cc.GodelCrdInformerFactory,
		cc.KatalystCrdInformerFactory,
		ctx.Done(),
		eventRecorder,
		cc.SchedulerConfig,
		godelrescheduler.WithReschedulerPolicyConfigs(cc.ReschedulerConfig.Profile.ReschedulerPolicyConfigs),
		godelrescheduler.WithDryRun(cc.DryRun),
	)
	if err != nil {
		return err
	}

	// Prepare the event broadcaster.
	cc.EventBroadcaster.StartRecordingToSink(ctx.Done())

	// Setup healthz checks.
	var checks []healthz.HealthChecker
	if *cc.ReschedulerConfig.LeaderElection.LeaderElect {
		checks = append(checks, cc.LeaderElection.WatchDog)
	}

	// Start up the healthz server.
	if cc.InsecureServing != nil {
		separateMetrics := cc.InsecureMetricsServing != nil
		handler := buildHandlerChain(newHealthzHandler(&cc.ReschedulerConfig, separateMetrics, checks...), nil, nil)
		if err := cc.InsecureServing.Serve(handler, 0, ctx.Done()); err != nil {
			return fmt.Errorf("failed to start healthz server: %v", err)
		}
	}
	if cc.InsecureMetricsServing != nil {
		handler := buildHandlerChain(newMetricsHandler(&cc.ReschedulerConfig), nil, nil)
		if err := cc.InsecureMetricsServing.Serve(handler, 0, ctx.Done()); err != nil {
			return fmt.Errorf("failed to start metrics server: %v", err)
		}
	}
	if cc.SecureServing != nil {
		handler := buildHandlerChain(newHealthzHandler(&cc.ReschedulerConfig, false, checks...), cc.Authentication.Authenticator, cc.Authorization.Authorizer)
		// TODO: handle stoppedCh returned by c.SecureServing.Serve
		if _, _, err := cc.SecureServing.Serve(handler, 0, ctx.Done()); err != nil {
			// fail early for secure handlers, removing the old error loop from above
			return fmt.Errorf("failed to start secure server: %v", err)
		}
	}

	// Start all informers.
	cc.InformerFactory.Start(ctx.Done())
	cc.GodelCrdInformerFactory.Start(ctx.Done())
	cc.KatalystCrdInformerFactory.Start(ctx.Done())

	// Wait for all caches to sync before scheduling.
	cc.InformerFactory.WaitForCacheSync(ctx.Done())
	cc.GodelCrdInformerFactory.WaitForCacheSync(ctx.Done())
	cc.KatalystCrdInformerFactory.WaitForCacheSync(ctx.Done())

	// If leader election is enabled, runCommand via LeaderElector until done and exit.
	if cc.LeaderElection != nil {
		cc.LeaderElection.Callbacks = leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				rescheduler.Run(ctx)
			},
			OnStoppedLeading: func() {
				select {
				case <-ctx.Done():
					// We were asked to terminate. Exit 0.
					klog.Info("Requested to terminate. Exiting.")
					os.Exit(0)
				default:
					// We lost the lock.
					klog.Exitf("leaderelection lost")
				}
			},
		}
		leaderElector, err := leaderelection.NewLeaderElector(*cc.LeaderElection)
		if err != nil {
			return fmt.Errorf("couldn't create leader elector: %v", err)
		}

		leaderElector.Run(ctx)

		return fmt.Errorf("lost lease")
	}

	// Leader election is disabled, so runCommand inline until done.
	rescheduler.Run(ctx)
	return fmt.Errorf("finished without leader elect")
}

// buildHandlerChain wraps the given handler with the standard filters.
func buildHandlerChain(handler http.Handler, authn authenticator.Request, authz authorizer.Authorizer) http.Handler {
	requestInfoResolver := &apirequest.RequestInfoFactory{}

	handler = genericapifilters.WithRequestInfo(handler, requestInfoResolver)
	handler = genericapifilters.WithCacheControl(handler)
	handler = genericfilters.WithPanicRecovery(handler, requestInfoResolver)

	return handler
}

func installMetricHandler(pathRecorderMux *mux.PathRecorderMux) {
	configz.InstallHandler(pathRecorderMux)
	//nolint:golint,ignore SA1019 See the Metrics Stability Migration KEP
	defaultMetricsHandler := legacyregistry.Handler().ServeHTTP
	pathRecorderMux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "DELETE" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			io.WriteString(w, "metrics reset\n")
			return
		}
		defaultMetricsHandler(w, req)
	})
}

// newMetricsHandler builds a metrics server from the config.
func newMetricsHandler(config *godelreschedulerconfig.GodelReschedulerConfiguration) http.Handler {
	pathRecorderMux := mux.NewPathRecorderMux(ComponentName)
	installMetricHandler(pathRecorderMux)
	if *config.EnableProfiling {
		routes.Profiling{}.Install(pathRecorderMux)
		if *config.EnableContentionProfiling {
			goruntime.SetBlockProfileRate(1)
		}
		routes.DebugFlags{}.Install(pathRecorderMux, "v", routes.StringFlagPutHandler(logs.GlogSetter))
	}
	return pathRecorderMux
}

// newHealthzHandler creates a healthz server from the config, and will also
// embed the metrics handler if the healthz and metrics address configurations
// are the same.
func newHealthzHandler(config *godelreschedulerconfig.GodelReschedulerConfiguration, separateMetrics bool, checks ...healthz.HealthChecker) http.Handler {
	pathRecorderMux := mux.NewPathRecorderMux(ComponentName)
	healthz.InstallHandler(pathRecorderMux, checks...)
	if !separateMetrics {
		installMetricHandler(pathRecorderMux)
	}
	if *config.EnableProfiling {
		routes.Profiling{}.Install(pathRecorderMux)
		if *config.EnableContentionProfiling {
			goruntime.SetBlockProfileRate(1)
		}
		routes.DebugFlags{}.Install(pathRecorderMux, "v", routes.StringFlagPutHandler(logs.GlogSetter))
	}
	return pathRecorderMux
}

func getEventRecorder(cc *reschedulerserverconfig.CompletedConfig) events.EventRecorder {
	return cc.EventBroadcaster.NewRecorder(ComponentName)
}
