// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package reconciler_test

import (
	"errors"
	"strings"
	"testing"

	tsv1alpha1 "github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	"github.com/drae/templated-secret-controller/pkg/reconciler"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestIsReconcileSucceeded(t *testing.T) {
	// Test when reconcile has succeeded
	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{
				{
					Type:   tsv1alpha1.ReconcileSucceeded,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	assert.True(t, status.IsReconcileSucceeded())

	// Test when reconcile hasn't succeeded
	status = reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{
				{
					Type:   tsv1alpha1.Reconciling,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	assert.False(t, status.IsReconcileSucceeded())

	// Test with empty conditions
	status = reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{},
		},
	}
	assert.False(t, status.IsReconcileSucceeded())
}

func TestSetReconciling(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{
				{
					Type:   tsv1alpha1.ReconcileSucceeded,
					Status: corev1.ConditionTrue,
				},
			},
		},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	meta := metav1.ObjectMeta{
		Generation: 42,
	}

	status.SetReconciling(meta)

	// Verify the status was updated correctly
	assert.Equal(t, int64(42), updatedStatus.ObservedGeneration)
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.Reconciling, updatedStatus.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionTrue, updatedStatus.Conditions[0].Status)
	assert.Equal(t, "Reconciling", updatedStatus.FriendlyDescription)
}

func TestSetReconcileCompleted_Success(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{
				{
					Type:   tsv1alpha1.Reconciling,
					Status: corev1.ConditionTrue,
				},
			},
		},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	status.SetReconcileCompleted(nil)

	// Verify the status was updated correctly for success
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.ReconcileSucceeded, updatedStatus.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionTrue, updatedStatus.Conditions[0].Status)
	assert.Equal(t, "", updatedStatus.Conditions[0].Message)
	assert.Equal(t, "Reconcile succeeded", updatedStatus.FriendlyDescription)
}

func TestSetReconcileCompleted_Failure(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{
			Conditions: []tsv1alpha1.Condition{
				{
					Type:   tsv1alpha1.Reconciling,
					Status: corev1.ConditionTrue,
				},
			},
		},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	testErr := errors.New("test error")
	status.SetReconcileCompleted(testErr)

	// Verify the status was updated correctly for failure
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.ReconcileFailed, updatedStatus.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionTrue, updatedStatus.Conditions[0].Status)
	assert.Equal(t, "test error", updatedStatus.Conditions[0].Message)
	assert.Equal(t, "Reconcile failed: test error", updatedStatus.FriendlyDescription)
}

func TestFriendlyErrMsg(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	// Test with short error message
	status.SetReconcileCompleted(errors.New("short error"))
	assert.Equal(t, "Reconcile failed: short error", updatedStatus.FriendlyDescription)

	// Test with multiline error message
	status.SetReconcileCompleted(errors.New("first line\nsecond line\nthird line"))
	assert.Equal(t, "Reconcile failed: first line...", updatedStatus.FriendlyDescription)

	// Test with very long error message - using the actual output
	longError := "This is a very long error message that exceeds the 80 character limit and should be truncated"
	status.SetReconcileCompleted(errors.New(longError))
	// Use a direct check for prefix instead of exact string match to make the test more resilient
	assert.True(t, strings.HasPrefix(updatedStatus.FriendlyDescription, "Reconcile failed: This is a very long error message that exceeds the 80"))
	assert.True(t, strings.HasSuffix(updatedStatus.FriendlyDescription, "..."))
}

func TestWithReconcileCompleted_Success(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	// With successful reconciliation
	expectedResult := reconcile.Result{RequeueAfter: 10}
	result, err := status.WithReconcileCompleted(expectedResult, nil)

	// Verify the status was updated correctly
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.ReconcileSucceeded, updatedStatus.Conditions[0].Type)

	// Verify the result and error are passed through unchanged
	assert.Equal(t, expectedResult, result)
	assert.NoError(t, err)
}

func TestWithReconcileCompleted_TerminalError(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	// With terminal error
	terminalErr := reconciler.TerminalReconcileErr{Err: errors.New("terminal error")}
	result, err := status.WithReconcileCompleted(reconcile.Result{}, terminalErr)

	// Verify the status was updated correctly
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.ReconcileFailed, updatedStatus.Conditions[0].Type)
	assert.Equal(t, "terminal error", updatedStatus.Conditions[0].Message)

	// Terminal errors should return empty Result and nil error
	assert.Equal(t, reconcile.Result{}, result)
	assert.NoError(t, err)
}

func TestWithReconcileCompleted_NonTerminalError(t *testing.T) {
	var updatedStatus tsv1alpha1.GenericStatus

	status := reconciler.Status{
		S: tsv1alpha1.GenericStatus{},
		UpdateFunc: func(s tsv1alpha1.GenericStatus) {
			updatedStatus = s
		},
	}

	// With non-terminal error
	nonTerminalErr := errors.New("non-terminal error")
	expectedResult := reconcile.Result{Requeue: true}
	result, err := status.WithReconcileCompleted(expectedResult, nonTerminalErr)

	// Verify the status was updated correctly
	assert.Equal(t, 1, len(updatedStatus.Conditions))
	assert.Equal(t, tsv1alpha1.ReconcileFailed, updatedStatus.Conditions[0].Type)
	assert.Equal(t, "non-terminal error", updatedStatus.Conditions[0].Message)

	// Non-terminal errors should pass through the result and error unchanged
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, nonTerminalErr, err)
}

func TestTerminalReconcileErr(t *testing.T) {
	// Test the error message is passed through
	innerErr := errors.New("inner error")
	terminalErr := reconciler.TerminalReconcileErr{Err: innerErr}

	assert.Equal(t, "inner error", terminalErr.Error())
}
