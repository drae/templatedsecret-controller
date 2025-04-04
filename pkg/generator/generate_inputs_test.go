// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"errors"
	"testing"

	"github.com/drae/templated-secret-controller/pkg/generator"
	"github.com/stretchr/testify/assert"
)

func TestAddFailsWithEmptyAnnotations(t *testing.T) {
	err := generator.GenerateInputs{}.Add(nil)
	assert.Equal(t, errors.New("internal inconsistency: called with annotations nil param"), err)
}

func TestAddSucceedsfulWithDefaultAnnotation(t *testing.T) {
	defaultAnnotations := map[string]string{
		"templatedsecret.starstreak.dev/generate-inputs": "",
	}
	err := generator.GenerateInputs{}.Add(defaultAnnotations)
	assert.Equal(t, nil, err)
}

// Tests for the IsChanged method
func TestIsChanged_NewInput(t *testing.T) {
	// Test when the annotation doesn't exist (should return true)
	inputs := generator.GenerateInputs{}.WithInputs(map[string]string{"key": "value"})
	anns := map[string]string{}
	
	isChanged := inputs.IsChanged(anns)
	
	assert.True(t, isChanged, "Should return true when annotation doesn't exist")
}

func TestIsChanged_DifferentInput(t *testing.T) {
	// Test when the annotation exists but with different value
	inputs := generator.GenerateInputs{}.WithInputs(map[string]string{"key": "new-value"})
	
	// Create annotations with a different serialized value
	anns := map[string]string{
		"templatedsecret.starstreak.dev/generate-inputs": `{"key":"old-value"}`,
	}
	
	isChanged := inputs.IsChanged(anns)
	
	assert.True(t, isChanged, "Should return true when inputs are different")
}

func TestIsChanged_SameInput(t *testing.T) {
	// Test when the annotation exists with the same value
	testInput := map[string]string{"key": "same-value"}
	inputs := generator.GenerateInputs{}.WithInputs(testInput)
	
	// Create annotations with the same serialized value
	anns := map[string]string{}
	err := generator.GenerateInputs{}.WithInputs(testInput).Add(anns)
	assert.NoError(t, err)
	
	isChanged := inputs.IsChanged(anns)
	
	assert.False(t, isChanged, "Should return false when inputs are the same")
}

func TestIsChanged_ComplexInput(t *testing.T) {
	// Test with a more complex input
	complexInput := map[string]interface{}{
		"string": "value",
		"number": 42,
		"nested": map[string]interface{}{
			"array": []string{"a", "b", "c"},
		},
	}
	
	inputs := generator.GenerateInputs{}.WithInputs(complexInput)
	
	// Create annotations with the same serialized value
	anns := map[string]string{}
	err := generator.GenerateInputs{}.WithInputs(complexInput).Add(anns)
	assert.NoError(t, err)
	
	isChanged := inputs.IsChanged(anns)
	
	assert.False(t, isChanged, "Should return false when complex inputs are the same")
}
