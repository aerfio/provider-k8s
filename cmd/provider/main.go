/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	"aerf.io/provider-k8s/apis"
	ctrls "aerf.io/provider-k8s/internal/controller"
)

type config struct {
	Debug            bool          `help:"Run with debug logging."`
	LeaderElection   bool          `help:"Use leader election for the controller manager."`
	PollInterval     time.Duration `help:"How often individual resources will be checked for drift from the desired state" default:"1m"`
	MaxReconcileRate int           `help:"The global maximum rate per second at which resources may checked for drift from the desired state." default:"10"`
}

func main() {
	cfg := config{}
	kctx := kong.Parse(&cfg, kong.DefaultEnvars("APP_"), kong.UsageOnError(), kong.Name("provider-k8s"))

	zl := zap.New(zap.UseDevMode(cfg.Debug))
	log := logging.NewLogrLogger(zl.WithName("provider-k8s"))
	if cfg.Debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	} else {
		ctrl.SetLogger(logr.Discard())
	}

	restCfg, err := ctrl.GetConfig()
	kctx.FatalIfErrorf(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(restCfg, cfg.MaxReconcileRate), ctrl.Options{
		// controller-runtime uses both ConfigMaps and Leases for leader
		// election by default. Leases expire after 15 seconds, with a
		// 10 second renewal deadline. We've observed leader loss due to
		// renewal deadlines being exceeded when under high load - i.e.
		// hundreds of reconciles per second and ~200rps to the API
		// server. Switching to Leases only and longer leases appears to
		// alleviate this.
		LeaderElection:             cfg.LeaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-k8s",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kctx.FatalIfErrorf(err, "Cannot create controller manager")
	kctx.FatalIfErrorf(apis.AddToScheme(mgr.GetScheme()), "Cannot add Object APIs to scheme")

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: cfg.MaxReconcileRate,
		PollInterval:            cfg.PollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(cfg.MaxReconcileRate),
		Features:                &feature.Flags{},
	}

	kctx.FatalIfErrorf(ctrls.Setup(mgr, o), "Cannot setup controllers")
	kctx.FatalIfErrorf(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
