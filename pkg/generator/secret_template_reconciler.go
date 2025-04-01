// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	tsv1alpha1 "github.com/drae/templatedsecret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/drae/templatedsecret-controller/pkg/client/clientset/versioned/scheme"
	"github.com/drae/templatedsecret-controller/pkg/reconciler"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	syncPeriod = 30 * time.Second
)

// ClientLoader allows Kubernetes Clients to be loaded from a Service Account.
type ClientLoader interface {
	Client(ctx context.Context, saName, saNamespace string) (client.Client, error)
}

// Tracker allows a tracking resource to track multiple other resources
type Tracker interface {
	Track(tracking types.NamespacedName, tracked ...types.NamespacedName)
	UntrackAll(tracking types.NamespacedName)
	GetTracking(tracked types.NamespacedName) []types.NamespacedName
}

// SecretTemplateReconciler watches for SecretTemplate Resources and generates a new secret from a set of input resources.
type SecretTemplateReconciler struct {
	mgr           manager.Manager
	client        client.Client
	saLoader      ClientLoader
	secretTracker Tracker
	log           logr.Logger
}

var (
	// Ensure SecretTemplateReconciler implements reconcile.Reconciler
	_ reconcile.Reconciler = &SecretTemplateReconciler{}
)

// NewSecretTemplateReconciler create a new SecretTemplate Reconciler
func NewSecretTemplateReconciler(mgr manager.Manager, client client.Client, loader ClientLoader, secretTracker Tracker, log logr.Logger) *SecretTemplateReconciler {
	return &SecretTemplateReconciler{mgr, client, loader, secretTracker, log}
}

// AttachWatches adds and starts watches this reconciler requires.
func (r *SecretTemplateReconciler) AttachWatches(c controller.Controller) error {
	// Watch for changes to created Secrets
	if err := c.Watch(
		source.Kind(
			r.mgr.GetCache(),
			&corev1.Secret{},
			handler.TypedEnqueueRequestForOwner[*corev1.Secret](r.mgr.GetScheme(), r.mgr.GetRESTMapper(), &tsv1alpha1.SecretTemplate{}, handler.OnlyControllerOwner()),
		),
	); err != nil {
		return err
	}

	err := c.Watch(
		source.Kind(
			r.mgr.GetCache(),
			&corev1.Secret{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, a *corev1.Secret) []reconcile.Request {
				var requests []reconcile.Request

				secretKey := types.NamespacedName{
					Namespace: a.GetNamespace(),
					Name:      a.GetName(),
				}

				for _, tracking := range r.secretTracker.GetTracking(secretKey) {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Name:      tracking.Name,
						Namespace: tracking.Namespace,
					}})
				}

				return requests
			}),
		),
	)

	if err != nil {
		return err
	}

	return c.Watch(
		source.Kind(
			r.mgr.GetCache(),
			&tsv1alpha1.SecretTemplate{},
			&handler.TypedEnqueueRequestForObject[*tsv1alpha1.SecretTemplate]{},
		),
	)
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *SecretTemplateReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)
	log.Info("reconciling")

	secretKey := types.NamespacedName{Namespace: request.Namespace, Name: request.Name}
	secretTemplate := tsv1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, secretKey, &secretTemplate); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")

			// Clear tracking if the SecretTemplate has been deleted.
			r.secretTracker.UntrackAll(secretKey)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if secretTemplate.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		S:          secretTemplate.Status.GenericStatus,
		UpdateFunc: func(st tsv1alpha1.GenericStatus) { secretTemplate.Status.GenericStatus = st },
	}

	status.SetReconciling(secretTemplate.ObjectMeta)
	defer r.updateStatus(ctx, &secretTemplate)

	return status.WithReconcileCompleted(r.reconcile(ctx, &secretTemplate))
}

func (r *SecretTemplateReconciler) reconcile(ctx context.Context, secretTemplate *tsv1alpha1.SecretTemplate) (reconcile.Result, error) {
	// Resolve input resources
	inputResources, err := r.resolveInputResources(ctx, secretTemplate)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Debug print of inputResources structure
	//	debugInputResources(inputResources)

	evaluatedTemplateSecret, err := evaluateTemplate(secretTemplate.Spec.JSONPathTemplate, inputResources)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create/Update Secret
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretTemplate.GetName(),
			Namespace: secretTemplate.GetNamespace(),
		},
	}

	if _, err = controllerutil.CreateOrUpdate(ctx, r.client, &secret, func() error {
		secret.Data = evaluatedTemplateSecret.Data
		secret.StringData = evaluatedTemplateSecret.StringData
		secret.ObjectMeta.Annotations = evaluatedTemplateSecret.Annotations
		secret.ObjectMeta.Labels = evaluatedTemplateSecret.Labels

		// Secret Type is immutable in Kubernetes, so if the template's type changes and
		// a secret already exists with a different type, we will need to delete and recreate
		// the secret. For now, we set the type, which will work for new secrets, and log a warning
		// if we detect a type change for existing secrets.
		if secret.Type != "" && secret.Type != evaluatedTemplateSecret.Type {
			r.log.Info("Warning: Secret type changes are not supported without manual deletion",
				"secret", secret.Name,
				"existingType", secret.Type,
				"desiredType", evaluatedTemplateSecret.Type)
		} else {
			secret.Type = evaluatedTemplateSecret.Type
		}

		return controllerutil.SetControllerReference(secretTemplate, &secret, scheme.Scheme)
	}); err != nil {
		return reconcile.Result{}, fmt.Errorf("creating/updating secret: %w", err)
	}

	secretTemplate.Status.Secret.Name = secret.Name

	// If not tracking input resources, periodically requeue
	if !shouldTrackInputResources(secretTemplate) {
		return reconcile.Result{RequeueAfter: syncPeriod}, nil
	}

	return reconcile.Result{}, nil
}

func (r *SecretTemplateReconciler) updateStatus(ctx context.Context, secretTemplate *tsv1alpha1.SecretTemplate) error {
	existingSecretTemplate := tsv1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: secretTemplate.Namespace, Name: secretTemplate.Name}, &existingSecretTemplate); err != nil {
		if errors.IsNotFound(err) {
			// The SecretTemplate was deleted after reconciliation started - this is not an error
			return nil
		}
		return fmt.Errorf("fetching secretTemplate: %w", err)
	}

	existingSecretTemplate.Status = secretTemplate.Status

	if err := r.client.Status().Update(ctx, &existingSecretTemplate); err != nil {
		if errors.IsConflict(err) {
			// Resource version changed - this will be handled on the next reconcile loop
			r.log.Info("Conflict detected when updating status, will retry on next reconcile",
				"secretTemplate", secretTemplate.Name)
			return nil
		}
		return fmt.Errorf("updating secretTemplate status: %w", err)
	}

	return nil
}

// Returns a client that was created using Service Account specified in the SecretTemplate spec.
// If no service account was specified then it returns the same Client as used by the SecretTemplateReconciler.
func (r *SecretTemplateReconciler) clientForSecretTemplate(ctx context.Context, secretTemplate *tsv1alpha1.SecretTemplate) (client.Client, error) {
	c := r.client
	if secretTemplate.Spec.ServiceAccountName != "" {
		saClient, err := r.saLoader.Client(ctx, secretTemplate.Spec.ServiceAccountName, secretTemplate.Namespace)
		if err != nil {
			return nil, err
		}
		c = saClient
	}
	return c, nil
}

func (r *SecretTemplateReconciler) resolveInputResources(ctx context.Context, secretTemplate *tsv1alpha1.SecretTemplate) (map[string]interface{}, error) {
	inputResourceclient, err := r.clientForSecretTemplate(ctx, secretTemplate)
	if err != nil {
		return nil, fmt.Errorf("unable to load client for reading Input Resources: %w", err)
	}

	secretTemplateKey := types.NamespacedName{Namespace: secretTemplate.Namespace, Name: secretTemplate.Name}
	resolvedInputResources := map[string]interface{}{}

	// Store resources to track in a local variable to avoid a race condition in the defer function
	var resolvedInputResourceKeys []types.NamespacedName

	// Cleanup function to ensure we track resources properly even in error cases
	defer func() {
		if shouldTrackInputResources(secretTemplate) {
			// Untrack everything first in case input resources have changed
			r.secretTracker.UntrackAll(secretTemplateKey)
			if len(resolvedInputResourceKeys) > 0 {
				r.secretTracker.Track(secretTemplateKey, resolvedInputResourceKeys...)
			}
		}
	}()

	for _, inputResource := range secretTemplate.Spec.InputResources {
		// Ensure we only load Secrets if using the default Client.
		if secretTemplate.Spec.ServiceAccountName == "" && (inputResource.Ref.Kind != "Secret" || inputResource.Ref.APIVersion != "v1") {
			return nil, fmt.Errorf("unable to load non-secrets without a specified serviceaccount")
		}

		unstructuredResource, err := resolveInputResource(inputResource.Ref, secretTemplate.Namespace, resolvedInputResources)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve input resource %s: %w", inputResource.Name, err)
		}

		key := types.NamespacedName{
			Namespace: secretTemplate.Namespace,
			Name:      unstructuredResource.GetName(),
		}

		if err := inputResourceclient.Get(ctx, key, &unstructuredResource); err != nil {
			return nil, fmt.Errorf("cannot fetch input resource %s: %w", unstructuredResource.GetName(), err)
		}

		content := unstructuredResource.UnstructuredContent()

		// If this is a Secret resource, decode base64 data fields immediately
		if unstructuredResource.GetKind() == "Secret" && unstructuredResource.GetAPIVersion() == "v1" {
			if data, exists := content["data"].(map[string]interface{}); exists {
				decodedData := map[string]interface{}{}
				for k, v := range data {
					if strVal, ok := v.(string); ok {
						decoded, err := base64.StdEncoding.DecodeString(strVal)
						if err != nil {
							return nil, fmt.Errorf("failed decoding base64 from Secret %s, data field %s: %w",
								unstructuredResource.GetName(), k, err)
						}
						decodedData[k] = string(decoded)
					}
				}
				// Replace the base64 encoded data with decoded values
				content["decodedData"] = decodedData
			}
		}

		resolvedInputResources[inputResource.Name] = content
		resolvedInputResourceKeys = append(resolvedInputResourceKeys, key)
	}

	return resolvedInputResources, nil
}

func resolveInputResource(ref tsv1alpha1.InputResourceRef, namespace string, inputResources map[string]interface{}) (unstructured.Unstructured, error) {
	// Only support jsonpath for Input Resource Reference Names.
	resolvedName, err := JSONPath(ref.Name).EvaluateWith(inputResources)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return toUnstructured(ref.APIVersion, ref.Kind, namespace, resolvedName.String())
}

// Returns whether we should track the resources contained in a SecretTemplate.
// We only track resources when a ServiceAccountName has not been specified. This implicitly means
// we only track Secret resources.
func shouldTrackInputResources(s *tsv1alpha1.SecretTemplate) bool {
	return s.Spec.ServiceAccountName == ""
}

func toUnstructured(apiVersion, kind, namespace, name string) (unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}

	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	obj.SetNamespace(namespace)

	return obj, nil
}

func evaluateTemplate(template *tsv1alpha1.JSONPathTemplate, values map[string]interface{}) (corev1.Secret, error) {
	// Check if template is nil to prevent panic
	if template == nil {
		return corev1.Secret{}, fmt.Errorf("JSONPathTemplate is nil")
	}

	// Template Secret Data
	data, err := evaluateBytes(template.Data, values)
	if err != nil {
		return corev1.Secret{}, fmt.Errorf("templating data: %w", err)
	}

	// Template Secret StringData
	stringData, err := evaluate(template.StringData, values)
	if err != nil {
		return corev1.Secret{}, fmt.Errorf("templating stringData: %w", err)
	}

	// Template Secret Annotations
	annotations, err := evaluate(template.Metadata.Annotations, values)
	if err != nil {
		return corev1.Secret{}, fmt.Errorf("templating annotations: %w", err)
	}

	// Template Secret Labels
	labels, err := evaluate(template.Metadata.Labels, values)
	if err != nil {
		return corev1.Secret{}, fmt.Errorf("templating labels: %w", err)
	}

	// Template Secret Type
	typeBuffer, err := JSONPath(template.Type).EvaluateWith(values)
	if err != nil {
		return corev1.Secret{}, fmt.Errorf("templating type: %w", err)
	}

	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretType(typeBuffer.String()),
		StringData: stringData,
		Data:       data,
	}, nil
}

func evaluate(mapping map[string]string, values map[string]interface{}) (map[string]string, error) {
	evaluatedMapping := map[string]string{}
	for key, expression := range mapping {
		valueBuffer, err := JSONPath(expression).EvaluateWith(values)
		if err != nil {
			return nil, err
		}

		exprStr := expression
		if strings.Contains(exprStr, ".data.") {
			// For paths like $(.resourceName.data.key), try using the already decoded data first
			decodedExpr := strings.Replace(exprStr, ".data.", ".decodedData.", 1)
			decodedValueBuffer, decodedErr := JSONPath(decodedExpr).EvaluateWith(values)
			if decodedErr == nil {
				// Successfully found data in decodedData, use it directly
				evaluatedMapping[key] = decodedValueBuffer.String()
				continue
			}
		}

		evaluatedMapping[key] = valueBuffer.String()
	}

	return evaluatedMapping, nil
}

func evaluateBytes(mapping map[string]string, values map[string]interface{}) (map[string][]byte, error) {
	evaluatedMapping := map[string][]byte{}
	for key, expression := range mapping {
		valueBuffer, err := JSONPath(expression).EvaluateWith(values)
		if err != nil {
			return nil, err
		}

		// Check if the expression refers to a path that might include decodedData
		// If it references a path that looks like .resourceName.data.field, try to convert it to use
		// decodedData instead to avoid decoding again
		exprStr := expression
		if strings.Contains(exprStr, ".data.") {
			// For paths like $(.resourceName.data.key), try using the already decoded data first
			decodedExpr := strings.Replace(exprStr, ".data.", ".decodedData.", 1)
			decodedValueBuffer, decodedErr := JSONPath(decodedExpr).EvaluateWith(values)
			if decodedErr == nil {
				// Successfully found data in decodedData, use it directly
				evaluatedMapping[key] = decodedValueBuffer.Bytes()
				continue
			}
		}

		// Fall back to the original behavior if decodedData approach doesn't work
		decoded, err := base64.StdEncoding.DecodeString(valueBuffer.String())
		if err != nil {
			return nil, fmt.Errorf("failed decoding base64 from a Secret: %w", err)
		}

		evaluatedMapping[key] = decoded
	}

	return evaluatedMapping, nil
}

func debugInputResources(inputResources map[string]interface{}) {
	// Print the inputResources map structure for debugging
	jsonData, err := json.MarshalIndent(inputResources, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling inputResources:", err)
		return
	}

	// Remove sensitive data from the output
	re := regexp.MustCompile(`"data":\s*{[^}]*}`)
	sanitizedJsonData := re.ReplaceAllString(string(jsonData), `"data": { ... }`)

	// Print the sanitized JSON data
	fmt.Println("Input Resources:")
	fmt.Println(strings.TrimSpace(sanitizedJsonData))
}
