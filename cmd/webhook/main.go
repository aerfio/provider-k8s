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
	"context"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type config struct {
	Debug bool `help:"Run with debug logging."`
}

func useColoredDevMode(enabled bool) zap.Opts {
	return func(opts *zap.Options) {
		if enabled {
			opts.EncoderConfigOptions = append(opts.EncoderConfigOptions, func(encoderConfig *zapcore.EncoderConfig) {
				encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			})
		}
		zap.UseDevMode(enabled)(opts)
	}
}

func main() {
	cfg := config{}
	kctx := kong.Parse(&cfg, kong.DefaultEnvars("APP_"), kong.UsageOnError(), kong.Name("provider-k8s-webhook"))

	zl := zap.New(useColoredDevMode(cfg.Debug))
	log := logging.NewLogrLogger(zl.WithName("provider-k8s"))
	ctrl.SetLogger(zl)
	//if cfg.Debug {
	//	// The controller-runtime runs with a no-op logger by default. It is
	//	// *very* verbose even at info level, so we only provide it a real
	//	// logger when we're running in debug mode.
	//	ctrl.SetLogger(zl)
	//} else {
	//	ctrl.SetLogger(logr.Discard())
	//}
	// Create a webhook server

	hookServer := webhook.DefaultServer{
		Options: webhook.Options{
			Port: 8443,
		},
	}
	mux := http.NewServeMux()
	mux.Handle("/livez", healthz.CheckHandler{
		Checker: healthz.Checker(func(req *http.Request) error {
			err := hookServer.StartedChecker()(req)
			log.Debug("livez", "err", err)
			return err
		}),
	})
	mux.Handle("/readyz", healthz.CheckHandler{
		Checker: healthz.Checker(func(req *http.Request) error {
			err := hookServer.StartedChecker()(req)
			log.Debug("readyz", "err", err)
			return err
		}),
	})
	hookServer.Options.WebhookMux = mux

	validatingHook := &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
			log.Info("req", "data", req)
			return webhook.Allowed("allowed")
		}),
	}
	hookServer.Register("/validate", validatingHook)

	// Start the server without a manger
	kctx.FatalIfErrorf(hookServer.Start(signals.SetupSignalHandler()))
}
