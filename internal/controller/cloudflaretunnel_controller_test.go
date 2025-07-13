/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/warjiang/cloudflare-tunnel-operator/api/v1alpha1"
)

var _ = Describe("CloudflareTunnel Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		cloudflaretunnel := &networkingv1alpha1.CloudflareTunnel{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CloudflareTunnel")
			err := k8sClient.Get(ctx, typeNamespacedName, cloudflaretunnel)
			if err != nil && errors.IsNotFound(err) {
				resource := &networkingv1alpha1.CloudflareTunnel{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: networkingv1alpha1.CloudflareTunnelSpec{
						Name:                resourceName,
						CloudflareAPIToken:  "test-token",
						CloudflareAccountID: "test-account-id",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &networkingv1alpha1.CloudflareTunnel{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CloudflareTunnel")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			mockCloudflareClient := new(MockCloudflareClient)
			controllerReconciler := &CloudflareTunnelReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				CloudflareClient: mockCloudflareClient,
			}

			// Mock the GetTunnelByName function to return a "not found" error,
			// which will trigger the tunnel creation logic.
			mockCloudflareClient.On("GetTunnelByName", ctx, "test-resource").Return(nil, fmt.Errorf("not found"))
			mockCloudflareClient.On("CreateTunnel", ctx, "test-resource").Return(&cloudflare.Tunnel{ID: "test-tunnel-id"}, []byte("test-secret"), nil)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify that the mock functions were called as expected
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})
	})
})
