// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCondition_Marshaling(t *testing.T) {
	// Test marshaling and unmarshaling of Condition type
	condition := v1alpha1.Condition{
		Type:    v1alpha1.ReconcileSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  "TestReason",
		Message: "Test message for the condition",
	}

	// Marshal to JSON
	bytes, err := json.Marshal(condition)
	assert.NoError(t, err, "Should marshal without error")
	assert.NotEmpty(t, bytes, "Marshaled bytes should not be empty")

	// Unmarshal from JSON
	var unmarshaled v1alpha1.Condition
	err = json.Unmarshal(bytes, &unmarshaled)
	assert.NoError(t, err, "Should unmarshal without error")

	// Verify fields were preserved
	assert.Equal(t, v1alpha1.ReconcileSucceeded, unmarshaled.Type)
	assert.Equal(t, corev1.ConditionTrue, unmarshaled.Status)
	assert.Equal(t, "TestReason", unmarshaled.Reason)
	assert.Equal(t, "Test message for the condition", unmarshaled.Message)
}

func TestGenericStatus_Marshaling(t *testing.T) {
	// Test marshaling and unmarshaling of GenericStatus type
	status := v1alpha1.GenericStatus{
		ObservedGeneration: 5,
		Conditions: []v1alpha1.Condition{
			{
				Type:    v1alpha1.Reconciling,
				Status:  corev1.ConditionTrue,
				Reason:  "Initializing",
				Message: "Starting reconciliation",
			},
			{
				Type:    v1alpha1.ReconcileSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "InProgress",
				Message: "Still reconciling",
			},
		},
		FriendlyDescription: "Reconciling resources",
	}

	// Marshal to JSON
	bytes, err := json.Marshal(status)
	assert.NoError(t, err, "Should marshal without error")
	assert.NotEmpty(t, bytes, "Marshaled bytes should not be empty")

	// Unmarshal from JSON
	var unmarshaled v1alpha1.GenericStatus
	err = json.Unmarshal(bytes, &unmarshaled)
	assert.NoError(t, err, "Should unmarshal without error")

	// Verify fields were preserved
	assert.Equal(t, int64(5), unmarshaled.ObservedGeneration)
	assert.Equal(t, 2, len(unmarshaled.Conditions))
	assert.Equal(t, v1alpha1.Reconciling, unmarshaled.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionTrue, unmarshaled.Conditions[0].Status)
	assert.Equal(t, "Initializing", unmarshaled.Conditions[0].Reason)
	assert.Equal(t, "Starting reconciliation", unmarshaled.Conditions[0].Message)
	assert.Equal(t, v1alpha1.ReconcileSucceeded, unmarshaled.Conditions[1].Type)
	assert.Equal(t, corev1.ConditionFalse, unmarshaled.Conditions[1].Status)
	assert.Equal(t, "InProgress", unmarshaled.Conditions[1].Reason)
	assert.Equal(t, "Still reconciling", unmarshaled.Conditions[1].Message)
	assert.Equal(t, "Reconciling resources", unmarshaled.FriendlyDescription)
}

func TestCondition_Types(t *testing.T) {
	// Test all defined condition types
	assert.Equal(t, v1alpha1.ConditionType("Reconciling"), v1alpha1.Reconciling)
	assert.Equal(t, v1alpha1.ConditionType("ReconcileFailed"), v1alpha1.ReconcileFailed)
	assert.Equal(t, v1alpha1.ConditionType("ReconcileSucceeded"), v1alpha1.ReconcileSucceeded)
	assert.Equal(t, v1alpha1.ConditionType("Invalid"), v1alpha1.Invalid)
}
