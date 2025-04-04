// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretTemplate_Marshaling(t *testing.T) {
	// Create a sample SecretTemplate
	template := v1alpha1.SecretTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretTemplate",
			APIVersion: "templatedsecret.starstreak.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Metadata: v1alpha1.SecretTemplateMetadata{
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"note": "test annotation",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": "admin",
			"password": "secret",
		},
		Spec: v1alpha1.SecretTemplateSpec{
			InputResources: []v1alpha1.InputResource{
				{
					Name: "input1",
					Ref: v1alpha1.InputResourceRef{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "source-secret",
					},
				},
			},
			JSONPathTemplate: &v1alpha1.JSONPathTemplate{
				StringData: map[string]string{
					"username": "$(.input1.data.username)",
					"password": "$(.input1.data.password)",
				},
				Type: corev1.SecretTypeOpaque,
				Metadata: v1alpha1.SecretTemplateMetadata{
					Labels: map[string]string{
						"generated": "true",
					},
				},
			},
			ServiceAccountName: "template-sa",
		},
		Status: v1alpha1.SecretTemplateStatus{
			Secret: corev1.LocalObjectReference{
				Name: "generated-secret",
			},
			GenericStatus: v1alpha1.GenericStatus{
				Conditions: []v1alpha1.Condition{
					{
						Type:   v1alpha1.ReconcileSucceeded,
						Status: corev1.ConditionTrue,
					},
				},
				FriendlyDescription: "Reconcile succeeded",
				ObservedGeneration:  1,
			},
			ObservedSecretResourceVersion: "12345",
		},
	}

	// Test JSON marshaling
	bytes, err := json.Marshal(template)
	assert.NoError(t, err, "Should marshal without error")
	assert.NotEmpty(t, bytes, "Marshaled bytes should not be empty")

	// Test JSON unmarshaling
	var unmarshaled v1alpha1.SecretTemplate
	err = json.Unmarshal(bytes, &unmarshaled)
	assert.NoError(t, err, "Should unmarshal without error")

	// Verify fields were preserved
	assert.Equal(t, "test-template", unmarshaled.Name)
	assert.Equal(t, "default", unmarshaled.Namespace)
	assert.Equal(t, "test", unmarshaled.Metadata.Labels["app"])
	assert.Equal(t, "test annotation", unmarshaled.Metadata.Annotations["note"])
	assert.Equal(t, corev1.SecretTypeOpaque, unmarshaled.Type)
	assert.Equal(t, "admin", unmarshaled.StringData["username"])
	assert.Equal(t, "secret", unmarshaled.StringData["password"])

	// Verify spec fields
	assert.Equal(t, 1, len(unmarshaled.Spec.InputResources))
	assert.Equal(t, "input1", unmarshaled.Spec.InputResources[0].Name)
	assert.Equal(t, "v1", unmarshaled.Spec.InputResources[0].Ref.APIVersion)
	assert.Equal(t, "Secret", unmarshaled.Spec.InputResources[0].Ref.Kind)
	assert.Equal(t, "source-secret", unmarshaled.Spec.InputResources[0].Ref.Name)
	assert.Equal(t, "template-sa", unmarshaled.Spec.ServiceAccountName)

	// Verify JSONPathTemplate
	assert.NotNil(t, unmarshaled.Spec.JSONPathTemplate)
	assert.Equal(t, "$(.input1.data.username)", unmarshaled.Spec.JSONPathTemplate.StringData["username"])
	assert.Equal(t, "$(.input1.data.password)", unmarshaled.Spec.JSONPathTemplate.StringData["password"])
	assert.Equal(t, "true", unmarshaled.Spec.JSONPathTemplate.Metadata.Labels["generated"])

	// Verify status fields
	assert.Equal(t, "generated-secret", unmarshaled.Status.Secret.Name)
	assert.Equal(t, 1, len(unmarshaled.Status.Conditions))
	assert.Equal(t, v1alpha1.ReconcileSucceeded, unmarshaled.Status.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionTrue, unmarshaled.Status.Conditions[0].Status)
	assert.Equal(t, "Reconcile succeeded", unmarshaled.Status.FriendlyDescription)
	assert.Equal(t, int64(1), unmarshaled.Status.ObservedGeneration)
	assert.Equal(t, "12345", unmarshaled.Status.ObservedSecretResourceVersion)
}

func TestSecretTemplateList_Marshaling(t *testing.T) {
	// Create a sample SecretTemplateList
	templateList := v1alpha1.SecretTemplateList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretTemplateList",
			APIVersion: "templatedsecret.starstreak.dev/v1alpha1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "v1",
		},
		Items: []v1alpha1.SecretTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "template-1",
					Namespace: "default",
				},
				Spec: v1alpha1.SecretTemplateSpec{
					InputResources: []v1alpha1.InputResource{
						{
							Name: "input1",
							Ref: v1alpha1.InputResourceRef{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "source-secret-1",
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "template-2",
					Namespace: "default",
				},
				Spec: v1alpha1.SecretTemplateSpec{
					InputResources: []v1alpha1.InputResource{
						{
							Name: "input1",
							Ref: v1alpha1.InputResourceRef{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "source-secret-2",
							},
						},
					},
				},
			},
		},
	}

	// Test JSON marshaling
	bytes, err := json.Marshal(templateList)
	assert.NoError(t, err, "Should marshal without error")
	assert.NotEmpty(t, bytes, "Marshaled bytes should not be empty")

	// Test JSON unmarshaling
	var unmarshaled v1alpha1.SecretTemplateList
	err = json.Unmarshal(bytes, &unmarshaled)
	assert.NoError(t, err, "Should unmarshal without error")

	// Verify fields were preserved
	assert.Equal(t, "SecretTemplateList", unmarshaled.Kind)
	assert.Equal(t, "templatedsecret.starstreak.dev/v1alpha1", unmarshaled.APIVersion)
	assert.Equal(t, "v1", unmarshaled.ResourceVersion)
	assert.Equal(t, 2, len(unmarshaled.Items))
	assert.Equal(t, "template-1", unmarshaled.Items[0].Name)
	assert.Equal(t, "template-2", unmarshaled.Items[1].Name)
	assert.Equal(t, "source-secret-1", unmarshaled.Items[0].Spec.InputResources[0].Ref.Name)
	assert.Equal(t, "source-secret-2", unmarshaled.Items[1].Spec.InputResources[0].Ref.Name)
}
