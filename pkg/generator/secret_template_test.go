// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	tsv1alpha1 "github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/drae/templated-secret-controller/pkg/client/clientset/versioned/scheme"
	"github.com/drae/templated-secret-controller/pkg/tracker"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/drae/templated-secret-controller/pkg/generator"
)

func Test_SecretTemplate(t *testing.T) {
	type test struct {
		name            string
		template        tsv1alpha1.SecretTemplate
		existingObjects []client.Object
		expectedSecret  corev1.Secret
	}

	tests := []test{
		{
			name: "reconciling secret template with input from another secret",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "existingSecret",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
							"key2": "$( .creds.data.inputKey2 )",
						},
						StringData: map[string]string{
							"key3": "value3",
						},
					},
				},
			},
			existingObjects: []client.Object{
				secret("existingSecret", map[string]string{
					"inputKey1": "value1",
					"inputKey2": "value2",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
				},
				Data: map[string][]byte{
					"key1": []byte("value1"),
					"key2": []byte("value2"),
				},
				StringData: map[string]string{
					"key3": "value3",
				},
			},
		},
		{
			name: "reconciling secret template with input from two inputs with dynamic inputname",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "first",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "first",
						},
					}, {
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "$( .first.data.secretName )",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("first", map[string]string{
					"secretName": "dynamic-secret-name",
				}),
				secret("dynamic-secret-name", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
				},
				Data: map[string][]byte{
					"key1": []byte("value1"),
				},
			},
		},
		{
			name: "reconciling secret template with embedded stringData template from configmap",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						StringData: map[string]string{
							"key1": "prefix-$(.map.data.inputKey1)-suffix",
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
				},
				StringData: map[string]string{
					"key1": "prefix-value1-suffix",
				},
			},
		},
		{
			name: "reconciling secret template with embedded stringData template in annotations",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Metadata: tsv1alpha1.SecretTemplateMetadata{
							Annotations: map[string]string{
								"annotation1": "$(.map.data.inputKey1)-suffix",
							},
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
					Annotations: map[string]string{
						"annotation1": "value1-suffix",
					},
				},
			},
		},
		{
			name: "reconciling secret template with embedded stringData template in labels",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Metadata: tsv1alpha1.SecretTemplateMetadata{
							Labels: map[string]string{
								"label1": "prefix-$(.map.data.inputKey1)",
							},
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
					Labels: map[string]string{
						"label1": "prefix-value1",
					},
				},
			},
		},
		{
			name: "reconciling secret template with embedded stringData template in type",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Type: corev1.SecretType("$(.map.data.inputKey1)"),
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "Opaque",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
				},
				Type: corev1.SecretType("Opaque"),
			},
		},
		{
			name: "reconciling secret template with type, annotations and labels",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "existingSecret",
						},
					}, {
						Name: "config",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingConfigMap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Type: "some-type",
						Metadata: tsv1alpha1.SecretTemplateMetadata{
							Labels: map[string]string{
								"label1": "$( .config.data.inputKey1 )",
							},
							Annotations: map[string]string{
								"annotation1": "$( .config.data.inputKey2 )",
							},
						},
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey3 )",
						},
						StringData: map[string]string{
							"key2": "$( .config.data.inputKey4 )",
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				secret("existingSecret", map[string]string{
					"inputKey3": "value3",
				}),
				configMap("existingConfigMap", map[string]string{
					"inputKey1": "value1",
					"inputKey2": "value2",
					"inputKey4": "value4",
				}),
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secretTemplate",
					Namespace:       "test",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						secretTemplateOwnerRef("secretTemplate"),
					},
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"annotation1": "value2",
					},
				},
				Type: "some-type",
				Data: map[string][]byte{
					"key1": []byte("value3"),
				},
				StringData: map[string]string{
					"key2": "value4",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allObjects := append(tc.existingObjects, &tc.template)
			secretTemplateReconciler, k8sClient := newReconciler(allObjects...)

			res, err := reconcileObject(t, secretTemplateReconciler, &tc.template)
			require.NoError(t, err)
			if tc.template.Spec.ServiceAccountName == "" {
				assert.Equal(t, 0*time.Second, res.RequeueAfter)
			} else {
				assert.Equal(t, 30*time.Second, res.RequeueAfter)
			}

			var secretTemplate tsv1alpha1.SecretTemplate
			err = k8sClient.Get(context.Background(), namespacedNameFor(&tc.template), &secretTemplate)
			require.NoError(t, err)

			assert.Equal(t, []tsv1alpha1.Condition{
				{Type: tsv1alpha1.ReconcileSucceeded, Status: corev1.ConditionTrue},
			}, secretTemplate.Status.Conditions)

			var actualSecret corev1.Secret
			err = k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      secretTemplate.Status.Secret.Name,
				Namespace: secretTemplate.GetNamespace(),
			}, &actualSecret)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedSecret, actualSecret)
			assert.Equal(t, secretTemplate.GetName(), secretTemplate.Status.Secret.Name, "reference secret name incorrect")
		})
	}
}

func Test_SecretTemplate_Errors(t *testing.T) {
	type test struct {
		name            string
		template        tsv1alpha1.SecretTemplate
		existingObjects []client.Object
		expectedError   string
	}

	tests := []test{
		{
			name: "reconciling secret template referencing non-existent resource",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "existingSecret",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
							"key2": "$( .creds.data.inputKey2 )",
						},
						StringData: map[string]string{
							"key3": "value3",
						},
					},
				},
			},
			expectedError: "cannot fetch input resource existingSecret: secrets \"existingSecret\" not found",
		},
		{
			name: "reconciling secret template referencing a resource with invalid apiversion",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					ServiceAccountName: "test",
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "//v1",
							Kind:       "ConfigMap",
							Name:       "existingConfigMap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
							"key2": "$( .creds.data.inputKey2 )",
						},
						StringData: map[string]string{
							"key3": "value3",
						},
					},
				},
			},
			expectedError: "unable to resolve input resource creds: unexpected GroupVersion string: //v1",
		},
		{
			name: "reconciling secret template with jsonpath that doesn't evaluate in data",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "existingSecret",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.doesntExist1 )",
						},
						StringData: map[string]string{
							"key3": "value3",
						},
					},
				},
			},
			existingObjects: []client.Object{
				secret("existingSecret", map[string]string{
					"inputKey1": "value1",
				}),
				secret("secretTemplate", map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				}),
			},
			expectedError: "templating data: doesntExist1 is not found",
		},
		{
			name: "reconciling secret template with jsonpath that doesn't evaluate in stringdata",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						StringData: map[string]string{
							"key1": "prefix-$(.map.data.doesntExist)-suffix",
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
				secret("secretTemplate", map[string]string{
					"key1": "prefix-value1-suffix",
				}),
			},
			expectedError: "templating stringData: doesntExist is not found",
		},
		{
			name: "reconciling secret template with jsonpath that doesn't evaluate in annotations",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Metadata: tsv1alpha1.SecretTemplateMetadata{
							Annotations: map[string]string{
								"key1": "prefix-$(.map.data.doesntExist)-suffix",
							},
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedError: "templating annotations: doesntExist is not found",
		},
		{
			name: "reconciling secret template with jsonpath that doesn't evaluate in labels",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Metadata: tsv1alpha1.SecretTemplateMetadata{
							Labels: map[string]string{
								"key1": "prefix-$(.map.data.doesntExist)-suffix",
							},
						},
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedError: "templating labels: doesntExist is not found",
		},
		{
			name: "reconciling secret template with jsonpath that doesn't evaluate in labels",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "map",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						Type: corev1.SecretType("$(.map.data.doesntExist)"),
					},
					ServiceAccountName: "service-account-client",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "Opaque",
				}),
			},
			expectedError: "templating type: doesntExist is not found",
		},
		{
			name: "reconciling secret template referencing non-secret without service account",
			template: tsv1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: tsv1alpha1.SecretTemplateSpec{
					InputResources: []tsv1alpha1.InputResource{{
						Name: "creds",
						Ref: tsv1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: &tsv1alpha1.JSONPathTemplate{
						StringData: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
						},
					},
					ServiceAccountName: "",
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedError: "unable to load non-secrets without a specified serviceaccount",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allObjects := append(tc.existingObjects, &tc.template)
			secretTemplateReconciler, k8sClient := newReconciler(allObjects...)

			_, err := reconcileObject(t, secretTemplateReconciler, &tc.template)
			require.Error(t, err)

			var secretTemplate tsv1alpha1.SecretTemplate
			err = k8sClient.Get(context.Background(), namespacedNameFor(&tc.template), &secretTemplate)
			require.NoError(t, err)

			assert.Equal(t, []tsv1alpha1.Condition{
				{Type: tsv1alpha1.ReconcileFailed, Status: corev1.ConditionTrue, Message: tc.expectedError},
			}, secretTemplate.Status.Conditions)

			var secret corev1.Secret
			err = k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      secretTemplate.Status.Secret.Name,
				Namespace: secretTemplate.GetNamespace(),
			}, &secret)
			require.Error(t, err)
		})
	}
}

func secret(name string, stringData map[string]string) *corev1.Secret {
	data := map[string][]byte{}

	for key, val := range stringData {
		data[key] = []byte(val)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Data: data,
	}
}

func configMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Data: data,
	}
}

func newReconciler(objects ...client.Object) (secretTemplateReconciler *generator.SecretTemplateReconciler, k8sClient client.Client) {
	tsv1alpha1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
	testLogr := zap.New(zap.UseDevMode(true))
	k8sClient = fakeClient.NewClientBuilder().WithObjects(objects...).WithScheme(scheme.Scheme).Build()

	// Create a fake manager that includes the cache
	fakeManager := &fakeManager{cache: &fakeCacheAdapter{client: k8sClient}}
	fakeClientLoader := fakeClientLoader{client: k8sClient}
	secretTemplateReconciler = generator.NewSecretTemplateReconciler(fakeManager, k8sClient, &fakeClientLoader, tracker.NewTracker(), testLogr)
	
	// Set max secret age to zero for test purposes
	// This ensures we don't requeue when ServiceAccountName is empty
	secretTemplateReconciler.SetReconciliationSettings(30*time.Second, 0)

	return secretTemplateReconciler, k8sClient
}

func reconcileObject(t *testing.T, recon *generator.SecretTemplateReconciler, object client.Object) (reconcile.Result, error) {
	res, err := recon.Reconcile(context.Background(), reconcile.Request{NamespacedName: namespacedNameFor(object)})
	require.False(t, res.Requeue)

	return res, err
}

func namespacedNameFor(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
}

func secretTemplateOwnerRef(name string) metav1.OwnerReference {
	truthy := true

	return metav1.OwnerReference{
		APIVersion:         "templatedsecret.starstreak.dev/v1alpha1",
		Kind:               "SecretTemplate",
		Name:               name,
		Controller:         &truthy,
		BlockOwnerDeletion: &truthy,
	}
}

// fakeClientLoader simply returns the same client for any Service Account
type fakeClientLoader struct {
	client client.Client
}

func (f *fakeClientLoader) Client(_ context.Context, _, _ string) (client.Client, error) {
	return f.client, nil
}

type fakeManager struct {
	manager manager.Manager
	cache   cache.Cache
}

// fakeCacheAdapter implements a minimal version of the cache.Cache interface for testing
type fakeCacheAdapter struct {
	client client.Client
}

func (f *fakeCacheAdapter) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return f.client.Get(ctx, key, obj, opts...)
}

func (f *fakeCacheAdapter) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return f.client.List(ctx, list, opts...)
}

func (f *fakeCacheAdapter) GetInformer(ctx context.Context, obj client.Object, opts ...cache.InformerGetOption) (cache.Informer, error) {
	// This method is not needed for our tests
	return nil, nil
}

func (f *fakeCacheAdapter) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind, opts ...cache.InformerGetOption) (cache.Informer, error) {
	// This method is not needed for our tests
	return nil, nil
}

func (f *fakeCacheAdapter) Start(ctx context.Context) error {
	// Nothing to start for our fake cache
	return nil
}

func (f *fakeCacheAdapter) WaitForCacheSync(ctx context.Context) bool {
	// Our fake cache is always synced
	return true
}

func (f *fakeCacheAdapter) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	// This method is not needed for our tests
	return nil
}

func (f *fakeCacheAdapter) NeedLeaderElection() bool {
	return false
}

func (f *fakeCacheAdapter) RemoveInformer(ctx context.Context, obj client.Object) error {
	// This method is not needed for our tests
	return nil
}

func (f *fakeManager) Add(_ manager.Runnable) error {
	return nil
}

func (f *fakeManager) Elected() <-chan struct{} {
	return nil
}

func (f *fakeManager) SetFields(_ interface{}) error {
	return nil
}

func (f *fakeManager) AddMetricsExtraHandler(_ string, _ http.Handler) error {
	return nil
}

func (f *fakeManager) AddMetricsServerExtraHandler(_ string, _ http.Handler) error {
	return nil
}

func (f *fakeManager) AddHealthzCheck(_ string, _ healthz.Checker) error {
	return nil
}

func (f *fakeManager) AddReadyzCheck(_ string, _ healthz.Checker) error {
	return nil
}

func (f *fakeManager) GetHTTPClient() *http.Client {
	return nil
}

func (f *fakeManager) Start(_ context.Context) error {
	return nil
}

func (f *fakeManager) GetConfig() *rest.Config {
	return nil
}

func (f *fakeManager) GetScheme() *runtime.Scheme {
	return scheme.Scheme
}

func (f *fakeManager) GetClient() client.Client {
	return nil
}

func (f *fakeManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (f *fakeManager) GetCache() cache.Cache {
	return f.cache
}

func (f *fakeManager) GetEventRecorderFor(_ string) record.EventRecorder {
	return nil
}

func (f *fakeManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (f *fakeManager) GetAPIReader() client.Reader {
	return nil
}

func (f *fakeManager) GetWebhookServer() webhook.Server {
	return nil
}

func (f *fakeManager) GetLogger() logr.Logger {
	return logr.Discard()
}

func (f *fakeManager) GetControllerOptions() config.Controller {
	return config.Controller{}
}

func (f *fakeManager) GetControllerNameAndOptions() (string, config.Controller) {
	return "", config.Controller{}
}
