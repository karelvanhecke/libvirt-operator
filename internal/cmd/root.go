/*
Copyright 2024 Karel Van Hecke

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

package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/controller"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"github.com/karelvanhecke/libvirt-operator/internal/webhook"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	defaultNamespacePath    = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	defaultLeaderElectionID = "lock.libvirt.karelvanhecke.com"
)

var (
	verbose          int
	logFormat        string
	namespace        string
	leaderElection   bool
	LeaderElectionID string
	metricsAddress   string
	probeAddress     string
)

func NewOperatorCmd() *cobra.Command {
	binaryName := filepath.Base(os.Args[0])

	cmd := &cobra.Command{
		Use: binaryName,
		Run: func(_ *cobra.Command, _ []string) { runOperator() },
	}

	cmd.Flags().CountVarP(&verbose, "verbose", "v", "set the log verbosity (-v up to -vvvv, -v [count] or --verbose [count])")
	cmd.Flags().StringVarP(&logFormat, "log-format", "f", "logfmt", "set the log format to json or logfmt")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "set the operator namespace")
	cmd.Flags().BoolVarP(&leaderElection, "leader-election", "l", true, "enable leader election")
	cmd.Flags().StringVarP(&LeaderElectionID, "leader-election-id", "i", defaultLeaderElectionID, "enable leader election")
	cmd.Flags().StringVarP(&metricsAddress, "metrics-address", "m", ":8080", "metrics address")
	cmd.Flags().StringVarP(&probeAddress, "probe-address", "p", ":8081", "probe address")

	cmd.AddCommand(newVersionCmd())

	return cmd
}

// +kubebuilder:webhook:path=/mutate-libvirt-karelvanhecke-com-v1alpha1-cloudinit,failurePolicy=fail,mutating=true,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=cloudinits,verbs=create,versions=v1alpha1,name=cloudinit-v1alpha1.libvirt.karelvanhecke.com
// +kubebuilder:webhook:path=/mutate-libvirt-karelvanhecke-com-v1alpha1-domain,failurePolicy=fail,mutating=true,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=domains,verbs=create,versions=v1alpha1,name=domains-v1alpha1.libvirt.karelvanhecke.com
// +kubebuilder:webhook:path=/mutate-libvirt-karelvanhecke-com-v1alpha1-volume,failurePolicy=fail,mutating=true,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=volumes,verbs=create,versions=v1alpha1,name=volumes-v1alpha1.libvirt.karelvanhecke.com

// +kubebuilder:webhook:path=/validate-libvirt-karelvanhecke-com-v1alpha1-cloudinit,failurePolicy=fail,mutating=false,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=cloudinits,verbs=create,versions=v1alpha1,name=cloudinit-v1alpha1.libvirt.karelvanhecke.com
// +kubebuilder:webhook:path=/validate-libvirt-karelvanhecke-com-v1alpha1-domain,failurePolicy=fail,mutating=false,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=domains,verbs=create,versions=v1alpha1,name=domains-v1alpha1.libvirt.karelvanhecke.com
// +kubebuilder:webhook:path=/validate-libvirt-karelvanhecke-com-v1alpha1-volume,failurePolicy=fail,mutating=false,admissionReviewVersions=v1,sideEffects=None,groups=libvirt.karelvanhecke.com,resources=volumes,verbs=create,versions=v1alpha1,name=volumes-v1alpha1.libvirt.karelvanhecke.com

// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=domains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=domains/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=domains/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pcidevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pcidevices/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pcidevices/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=networks/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=cloudinits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=cloudinits/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=cloudinits/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=volumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=volumes/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=volumes/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pools/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=pools/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=hosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=hosts/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=hosts/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=auths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=auths/status,verbs=get;update;patch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=libvirt.karelvanhecke.com,resources=auths/finalizers,verbs=update
// +kubebuilder:rbac:namespace=libvirt-operator,groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:namespace=libvirt-operator,groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=libvirt-operator,groups="",resources=events,verbs=create;patch

func runOperator() {
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.Level(-(verbose)))

	handlerOpts := &slog.HandlerOptions{Level: levelVar}
	var handler slog.Handler
	handler = slog.NewTextHandler(os.Stderr, handlerOpts)

	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, handlerOpts)
	}

	logger := logr.FromSlogHandler(handler)
	setupLog := logger.WithName("setup")
	klog.SetLogger(logger)
	ctrl.SetLogger(logger)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "failed to get kubeconfig")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "failed to add scheme")
		os.Exit(1)
	}

	if err := v1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "failed to add scheme libvirt.karelvanhecke.com/v1alpha1")
		os.Exit(1)
	}

	if namespace == "" {
		ns, err := os.ReadFile(defaultNamespacePath)
		if err != nil {
			setupLog.Error(err, "could not determine operator namespace")
			os.Exit(1)
		}
		namespace = string(ns)
	}

	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		LeaderElection:   leaderElection,
		LeaderElectionID: LeaderElectionID,
		Metrics: server.Options{
			BindAddress: metricsAddress,
		},
		HealthProbeBindAddress: probeAddress,
	})
	if err != nil {
		setupLog.Error(err, "failed to create manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to setup healthz check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to setup readyz check")
		os.Exit(1)
	}

	authStore := store.NewAuthStore()
	hostStore := store.NewHostStore()

	if err := (&controller.AuthReconciler{
		Client:    mgr.GetClient(),
		AuthStore: authStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup auth controller with manager")
		os.Exit(1)
	}

	if err := (&controller.HostReconciler{
		Client:    mgr.GetClient(),
		AuthStore: authStore,
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup host controller with manager")
		os.Exit(1)
	}

	if err := (&controller.PoolReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup pool controller with manager")
		os.Exit(1)
	}

	if err := (&controller.NetworkReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup network controller with manager")
		os.Exit(1)
	}

	if err := (&controller.PCIDeviceReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup pci device controller with manager")
		os.Exit(1)
	}

	if err := (&controller.VolumeReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup volume controller with manager")
		os.Exit(1)
	}

	if err := (&controller.CloudInitReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup cloud-init controller with manager")
		os.Exit(1)
	}

	if err := (&controller.DomainReconciler{
		Client:    mgr.GetClient(),
		HostStore: hostStore,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup domain controller with manager")
		os.Exit(1)
	}

	if err := webhook.SetupVolumeWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup volume webhook with manager")
		os.Exit(1)
	}

	if err := webhook.SetupCloudInitWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup cloud-init webhook with manager")
		os.Exit(1)
	}

	if err := webhook.SetupDomainWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup domain webhook with manager")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "failed to start manager")
		os.Exit(1)
	}
}
