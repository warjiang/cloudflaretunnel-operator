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
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/warjiang/cloudflaretunnel-operator/api/v1alpha1"
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
						Name: resourceName,
						CredentialsRef: networkingv1alpha1.CredentialsSecretRef{
							Name: "cloudflare-credentials",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			tokenSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-token", resourceName), Namespace: "default"}}
			_ = k8sClient.Delete(ctx, tokenSecret)
			connectorDeployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-connector", resourceName), Namespace: "default"}}
			_ = k8sClient.Delete(ctx, connectorDeployment)

			resource := &networkingv1alpha1.CloudflareTunnel{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if errors.IsNotFound(err) {
				return
			}
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
			mockCloudflareClient.On("GetTunnelTokenByID", ctx, "test-tunnel-id").Return("test-token", nil)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			tokenSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-resource-token", Namespace: "default"}, tokenSecret)).To(Succeed())
			Expect(tokenSecret.Data).To(HaveKeyWithValue("token", []byte("test-token")))

			connectorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-resource-connector", Namespace: "default"}, connectorDeployment)).To(Succeed())
			Expect(connectorDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(connectorDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal("cloudflare/cloudflared:2026.3.0"))

			// Verify that the mock functions were called as expected
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should delete connector pods before deleting tunnel", func() {
			mockCloudflareClient := new(MockCloudflareClient)
			controllerReconciler := &CloudflareTunnelReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				CloudflareClient: mockCloudflareClient,
			}

			resource := &networkingv1alpha1.CloudflareTunnel{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			resource.Finalizers = append(resource.Finalizers, finalizer)
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			resource.Status.TunnelID = "test-tunnel-id"
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			labels := connectorLabels(resource)
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      desiredConnectorDeploymentName(resource),
					Namespace: resource.Namespace,
					Labels:    labels,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: labels},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: labels},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "cloudflared", Image: "cloudflare/cloudflared:2026.3.0"}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			mockCloudflareClient.AssertNotCalled(GinkgoT(), "DeleteTunnelByID", mock.Anything, "test-tunnel-id")

			mockCloudflareClient.On("DeleteTunnelByID", ctx, "test-tunnel-id").Return(nil).Once()
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})
	})
})
