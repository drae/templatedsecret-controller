// Copyright 2024 The Templatedsecret Controller Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tsv1alpha1 "github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/drae/templated-secret-controller/pkg/generator"
	"github.com/drae/templated-secret-controller/pkg/satoken"
	"github.com/drae/templated-secret-controller/pkg/tracker"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
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
	// Version of templated-secret-controller is set via ldflags at build-time from most recent git tag
	Version = "develop"

	log                        = logf.Log.WithName("ts")
	ctrlNamespace              = ""
	watchNamespaces            = ""
	metricsBindAddress         = ""
	enableLeaderElection       = false
	leaderElectionResourceName = "templated-secret-controller-leader-election"
	reconciliationInterval     = time.Hour
	maxSecretAge               = 720 * time.Hour
	logLevel                   = "info"
)

func main() {
	flag.StringVar(&ctrlNamespace, "namespace", "", "Namespace to watch (deprecated, use --watch-namespaces instead)")
	flag.StringVar(&watchNamespaces, "watch-namespaces", "", "Comma-separated list of namespaces to watch (empty for all)")
	flag.StringVar(&metricsBindAddress, "metrics-bind-address", ":8080", "Address for metrics server. If 0, then metrics server doesn't listen on any port.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager")
	flag.StringVar(&leaderElectionResourceName, "leader-election-id", "templated-secret-controller-leader-election", "Resource name for leader election")
	flag.DurationVar(&reconciliationInterval, "reconciliation-interval", time.Hour, "How often to reconcile SecretTemplates")
	flag.DurationVar(&maxSecretAge, "max-secret-age", 720*time.Hour, "Maximum age of a secret before forcing regeneration")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Set up zap logger with configured log level
	opts := zap.Options{
		Development: false,
	}

	// Configure log level
	switch strings.ToLower(logLevel) {
	case "debug":
		opts.Development = true
	case "info":
		// Default level
	case "warn", "warning":
		// Note: zap Options doesn't have a direct way to set log level through Options
		// This would need a custom zapcore level configuration
	case "error":
		// Note: zap Options doesn't have a direct way to set log level through Options
		// This would need a custom zapcore level configuration
	default:
		fmt.Fprintf(os.Stderr, "Unsupported log level: %s. Using 'info'\n", logLevel)
	}

	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	entryLog := log.WithName("entrypoint")
	entryLog.Info("templated-secret-controller", "version", Version)

	entryLog.Info("setting up manager")
	restConfig := config.GetConfigOrDie()

	// Register API types
	tsv1alpha1.AddToScheme(scheme.Scheme)

	// Wait for CRDs to be ready before starting controller
	entryLog.Info("waiting for SecretTemplate CRD to be ready")
	exitIfErr(entryLog, "waiting for CRDs", waitForCRDs(restConfig, entryLog))

	// Setup manager options
	recoverPanic := true

	// Handle namespace configuration - support both legacy and new approach
	namespaces := make(map[string]cache.Config)
	if watchNamespaces != "" {
		// New approach: watch multiple namespaces
		for _, ns := range strings.Split(watchNamespaces, ",") {
			if ns = strings.TrimSpace(ns); ns != "" {
				namespaces[ns] = cache.Config{}
			}
		}
	} else if ctrlNamespace != "" {
		// Legacy approach: single namespace
		namespaces[ctrlNamespace] = cache.Config{}
	}

	managerOptions := manager.Options{
		// Use proper namespace selector field in newer controller-runtime
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
		},
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
		// Configure leader election
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        leaderElectionResourceName,
		LeaderElectionNamespace: "", // Use controller namespace if empty
	}

	// Add controller-specific options for newer versions of controller-runtime
	managerOptions.Controller.RecoverPanic = &recoverPanic

	mgr, err := manager.New(restConfig, managerOptions)
	exitIfErr(entryLog, "unable to set up controller manager", err)

	entryLog.Info("setting up controller")

	coreClient, err := kubernetes.NewForConfig(restConfig)
	exitIfErr(entryLog, "building core client", err)

	saLoader := generator.NewServiceAccountLoader(satoken.NewManager(coreClient, log.WithName("template")))

	// Set SecretTemplate's maximum exponential to reduce reconcile time for inputresource errors
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, 120*time.Second)
	secretTemplateReconciler := generator.NewSecretTemplateReconciler(mgr, mgr.GetClient(), saLoader, tracker.NewTracker(), log.WithName("template"))

	// Pass reconciliation settings to the reconciler
	secretTemplateReconciler.SetReconciliationSettings(reconciliationInterval, maxSecretAge)
	entryLog.Info("configured reconciliation settings",
		"interval", reconciliationInterval.String(),
		"maxSecretAge", maxSecretAge.String())

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

func waitForCRDs(restConfig *rest.Config, entryLog logr.Logger) error {
	apiExtClient, err := apiextensionsclientset.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("unable to create API extensions client: %w", err)
	}

	// First check that the CRD exists and is established
	err = wait.PollImmediate(1*time.Second, 60*time.Second, func() (bool, error) {
		crd, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "secrettemplates.templatedsecret.starstreak.dev", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				entryLog.Info("SecretTemplate CRD not found, retrying...")
				return false, nil
			}
			return false, err
		}

		// Check that the CRD is established
		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1.Established &&
				condition.Status == apiextensionsv1.ConditionTrue {
				entryLog.Info("SecretTemplate CRD is established")
				return true, nil
			}
		}

		entryLog.Info("SecretTemplate CRD found but not yet established, retrying...")
		return false, nil
	})

	if err != nil {
		return err
	}

	// Now verify that the API resource is actually discoverable
	// This ensures the apiserver's discovery cache has been updated
	discoveryClient := apiExtClient.Discovery()
	entryLog.Info("Verifying API resource is discoverable")

	return wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		resourceList, err := discoveryClient.ServerResourcesForGroupVersion("templatedsecret.starstreak.dev/v1alpha1")
		if err != nil {
			entryLog.Info("API resource not yet discoverable, waiting for API server discovery cache to refresh...",
				"error", err.Error())
			return false, nil
		}

		for _, r := range resourceList.APIResources {
			if r.Kind == "SecretTemplate" {
				entryLog.Info("SecretTemplate API resource is now discoverable")
				return true, nil
			}
		}

		entryLog.Info("SecretTemplate kind not found in API resources, waiting...")
		return false, nil
	})
}
