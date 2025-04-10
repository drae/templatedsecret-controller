//
// Original source - secretgen-controller - Copyright 2024 The Carvel Authors.
// Re-organized and updated as - templated-secret-controller - (C) 2025 starstreak.dev
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/drae/templated-secret-controller/pkg/apis/templatedsecret/v1alpha1"
	templatedsecretv1alpha1 "github.com/drae/templated-secret-controller/pkg/client/clientset/versioned/typed/templatedsecret/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeSecretTemplates implements SecretTemplateInterface
type fakeSecretTemplates struct {
	*gentype.FakeClientWithList[*v1alpha1.SecretTemplate, *v1alpha1.SecretTemplateList]
	Fake *FakeTemplatedsecretV1alpha1
}

func newFakeSecretTemplates(fake *FakeTemplatedsecretV1alpha1, namespace string) templatedsecretv1alpha1.SecretTemplateInterface {
	return &fakeSecretTemplates{
		gentype.NewFakeClientWithList[*v1alpha1.SecretTemplate, *v1alpha1.SecretTemplateList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("secrettemplates"),
			v1alpha1.SchemeGroupVersion.WithKind("SecretTemplate"),
			func() *v1alpha1.SecretTemplate { return &v1alpha1.SecretTemplate{} },
			func() *v1alpha1.SecretTemplateList { return &v1alpha1.SecretTemplateList{} },
			func(dst, src *v1alpha1.SecretTemplateList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.SecretTemplateList) []*v1alpha1.SecretTemplate {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.SecretTemplateList, items []*v1alpha1.SecretTemplate) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
