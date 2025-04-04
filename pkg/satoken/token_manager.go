// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// This file is a modified version of
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/token/token_manager.go

// Package satoken provides a Service Account Token getter and cache for kubernetes
// ServiceAccount tokens.
package satoken

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/drae/templated-secret-controller/pkg/generator"
	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"
)

const (
	maxTTL    = 2 * time.Hour
	gcPeriod  = time.Hour * 24
	maxJitter = 10 * time.Second
)

var _ generator.TokenManager = &Manager{}

// NewManager returns a new token manager.
func NewManager(c clientset.Interface, log logr.Logger) *Manager {
	m := &Manager{
		getToken: func(ctx context.Context, name, namespace string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
			return c.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, tr, metav1.CreateOptions{})
		},
		reviewToken: func(ctx context.Context, tr *authenticationv1.TokenReview) (*authenticationv1.TokenReview, error) {
			return c.AuthenticationV1().TokenReviews().Create(ctx, tr, metav1.CreateOptions{})
		},
		cache: make(map[string]*authenticationv1.TokenRequest),
		clock: clock.RealClock{},
		log:   log,
	}
	go wait.Forever(m.cleanup, gcPeriod)
	return m
}

// Manager manages service account tokens for pods.
type Manager struct {

	// cacheMutex guards the cache
	cacheMutex sync.RWMutex
	cache      map[string]*authenticationv1.TokenRequest

	// mocked for testing
	getToken    func(ctx context.Context, name, namespace string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
	reviewToken func(ctx context.Context, tr *authenticationv1.TokenReview) (*authenticationv1.TokenReview, error)
	clock       clock.Clock

	log logr.Logger
}

// GetServiceAccountToken gets a service account token from cache or
// from the TokenRequest API. This process is as follows:
// * Check the cache for the current token request.
// * If the token exists and does not require a refresh, return the current token.
// * Attempt to refresh the token.
// * If the token is refreshed successfully, save it in the cache and return the token.
// * If refresh fails and the old token is still valid, log an error and return the old token.
// * If refresh fails and the old token is no longer valid, return an error
func (m *Manager) GetServiceAccountToken(ctx context.Context, namespace, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	key := fmt.Sprintf("%q/%q", name, namespace)

	ctr, ok := m.get(key)

	if ok && !m.requiresRefresh(ctx, ctr) {
		return ctr, nil
	}

	tr, err := m.getToken(ctx, name, namespace, tr)
	if err != nil {
		switch {
		case !ok:
			return nil, fmt.Errorf("Fetch token: %v", err)
		case m.expired(ctr):
			return nil, fmt.Errorf("Token %s expired and refresh failed: %v", key, err)
		default:
			m.log.Error(err, "Update token", "cacheKey", key)
			return ctr, nil
		}
	}

	m.set(key, tr)
	return tr, nil
}

func (m *Manager) cleanup() {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	for k, tr := range m.cache {
		if m.expired(tr) {
			delete(m.cache, k)
		}
	}
}

func (m *Manager) get(key string) (*authenticationv1.TokenRequest, bool) {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()
	ctr, ok := m.cache[key]
	return ctr, ok
}

func (m *Manager) set(key string, tr *authenticationv1.TokenRequest) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	m.cache[key] = tr
}

func (m *Manager) expired(t *authenticationv1.TokenRequest) bool {
	return m.clock.Now().After(t.Status.ExpirationTimestamp.Time)
}

// requiresRefresh returns true if the token is older half of it's maxTTL
func (m *Manager) requiresRefresh(ctx context.Context, tr *authenticationv1.TokenRequest) bool {
	review, err := m.reviewToken(ctx, &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: tr.Status.Token,
		},
	})
	if err != nil || !review.Status.Authenticated {
		return true
	}

	if tr.Spec.ExpirationSeconds == nil {
		cpy := tr.DeepCopy()
		cpy.Status.Token = ""
		m.log.Info("Expiration seconds was nil for token request", "tokenRequest", cpy)
		return false
	}
	now := m.clock.Now()
	exp := tr.Status.ExpirationTimestamp.Time
	iat := exp.Add(-1 * time.Duration(*tr.Spec.ExpirationSeconds) * time.Second)

	jitter := time.Duration(rand.Float64()*maxJitter.Seconds()) * time.Second
	if now.After(iat.Add(maxTTL - jitter)) {
		return true
	}

	// Require a refresh if within 50% of the TTL plus a jitter from the expiration time.
	return now.After(exp.Add(-1*time.Duration((*tr.Spec.ExpirationSeconds*50)/100)*time.Second - jitter))
}
