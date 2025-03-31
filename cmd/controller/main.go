// Copyright 2024 The Templatedsecret Controller Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tsv1alpha1 "github.com/drae/templatedsecret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/drae/templatedsecret-controller/pkg/generator"
	"github.com/drae/templatedsecret-controller/pkg/satoken"
	"github.com/drae/templatedsecret-controller/pkg/tracker"

	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	// Version of templatedsecret-controller is set via ldflags at build-time from most recent git tag
	Version = "develop"

	log                = logf.Log.WithName("ts")
	ctrlNamespace      = ""
	metricsBindAddress = ""
)

func main() {
	flag.StringVar(&ctrlNamespace, "namespace", "", "Namespace to watch")
	flag.StringVar(&metricsBindAddress, "metrics-bind-address", ":8080", "Address for metrics server. If 0, then metrics server doesnt listen on any port.")
	flag.Parse()

	opts := zap.Options{
		Development: false,
	}
	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	entryLog := log.WithName("entrypoint")
	entryLog.Info("templatedsecret-controller", "version", Version)

	entryLog.Info("setting up manager")
	restConfig := config.GetConfigOrDie()

	// Register API types
	tsv1alpha1.AddToScheme(scheme.Scheme)

	// Setup manager options
	recoverPanic := true
	managerOptions := manager.Options{
		// Use proper namespace selector field in newer controller-runtime
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				ctrlNamespace: {},
			},
		},
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
	}

	// Add controller-specific options for newer versions of controller-runtime
	managerOptions.Controller.RecoverPanic = &recoverPanic

	mgr, err := manager.New(restConfig, managerOptions)
	exitIfErr(entryLog, "unable to set up controller manager", err)

	entryLog.Info("setting up controller")

	coreClient, err := kubernetes.NewForConfig(restConfig)
	exitIfErr(entryLog, "building core client", err)

	// tsClient is currently unused. Uncomment the following lines if needed in the future.
	// tsClient, err := tsclient.NewForConfig(restConfig)
	// exitIfErr(entryLog, "building templatedsecret client", err)

	saLoader := generator.NewServiceAccountLoader(satoken.NewManager(coreClient, log.WithName("template")))

	// Set SecretTemplate's maximum exponential to reduce reconcile time for inputresource errors
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, 120*time.Second)
	secretTemplateReconciler := generator.NewSecretTemplateReconciler(mgr.GetClient(), saLoader, tracker.NewTracker(), log.WithName("template"))
	exitIfErr(entryLog, "registering", registerCtrlWithRateLimiter("template", mgr, secretTemplateReconciler, rateLimiter))

	entryLog.Info("starting manager")

	err = mgr.Start(signals.SetupSignalHandler())
	exitIfErr(entryLog, "unable to run manager", err)
}

type reconcilerWithWatches interface {
	reconcile.Reconciler
	AttachWatches(controller.Controller) error
}

func registerCtrlWithRateLimiter(desc string, mgr manager.Manager, reconciler reconcilerWithWatches, rateLimiter workqueue.RateLimiter) error {
	ctrlName := "ts-" + desc

	// Create a custom adapter for the RateLimiter
	adaptedRateLimiter := &typedRateLimiterAdapter{
		RateLimiter: rateLimiter,
	}

	ctrlOpts := controller.Options{
		Reconciler: reconciler,
		// Default MaxConcurrentReconciles is 1. Keeping at that
		// since we are not doing anything that we need to parallelize for.
		RateLimiter: adaptedRateLimiter,
	}

	ctrl, err := controller.New(ctrlName, mgr, ctrlOpts)
	if err != nil {
		return fmt.Errorf("%s: unable to set up: %s", ctrlName, err)
	}

	err = reconciler.AttachWatches(ctrl)
	if err != nil {
		return fmt.Errorf("%s: unable to attaches watches: %s", ctrlName, err)
	}

	return nil
}

// typedRateLimiterAdapter adapts a generic RateLimiter to be used with specific types
type typedRateLimiterAdapter struct {
	workqueue.RateLimiter
}

// When implements TypedRateLimiter.When
func (a *typedRateLimiterAdapter) When(item reconcile.Request) time.Duration {
	return a.RateLimiter.When(item)
}

// Forget implements TypedRateLimiter.Forget
func (a *typedRateLimiterAdapter) Forget(item reconcile.Request) {
	a.RateLimiter.Forget(item)
}

// NumRequeues implements TypedRateLimiter.NumRequeues
func (a *typedRateLimiterAdapter) NumRequeues(item reconcile.Request) int {
	return a.RateLimiter.NumRequeues(item)
}

func exitIfErr(entryLog logr.Logger, desc string, err error) {
	if err != nil {
		entryLog.Error(err, desc)
		os.Exit(1)
	}
}
