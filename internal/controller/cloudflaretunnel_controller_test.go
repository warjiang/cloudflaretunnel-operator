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
		const tunnelID = "test-tunnel-id"

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
			connectorConfigMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-connector-config", resourceName), Namespace: "default"}}
			_ = k8sClient.Delete(ctx, connectorConfigMap)
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
			mockCloudflareClient.On("CreateTunnel", ctx, "test-resource").Return(&cloudflare.Tunnel{ID: tunnelID}, []byte("test-secret"), nil)
			mockCloudflareClient.On("GetTunnelTokenByID", ctx, tunnelID).Return("test-token", nil)

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

		It("should reconcile ingress rules into configmap and deployment args", func() {
			mockCloudflareClient := new(MockCloudflareClient)
			controllerReconciler := &CloudflareTunnelReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				CloudflareClient: mockCloudflareClient,
			}

			ingressResourceName := "test-ingress-resource"
			ingressNamespacedName := types.NamespacedName{
				Name:      ingressResourceName,
				Namespace: "default",
			}
			resource := &networkingv1alpha1.CloudflareTunnel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ingressResourceName,
					Namespace: "default",
				},
				Spec: networkingv1alpha1.CloudflareTunnelSpec{
					Name:     ingressResourceName,
					Hostname: "karmada.example.com",
					ZoneID:   "zone-id-123",
					CredentialsRef: networkingv1alpha1.CredentialsSecretRef{
						Name: "cloudflare-credentials",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			resource.Spec.Ingress = &networkingv1alpha1.IngressSpec{
				Rules: []networkingv1alpha1.IngressRule{
					{
						Path: "/api",
						Service: networkingv1alpha1.IngressServiceBackend{
							Name:      "backend-svc",
							Namespace: "backend-ns",
							Port:      8080,
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, resource)
				_ = k8sClient.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-token", ingressResourceName), Namespace: "default"}})
				_ = k8sClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-connector-config", ingressResourceName), Namespace: "default"}})
				_ = k8sClient.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-connector", ingressResourceName), Namespace: "default"}})
			}()

			mockCloudflareClient.On("GetTunnelByName", ctx, ingressResourceName).Return(nil, fmt.Errorf("not found")).Once()
			mockCloudflareClient.On("CreateTunnel", ctx, ingressResourceName).Return(&cloudflare.Tunnel{ID: tunnelID}, []byte("test-secret"), nil).Once()
			mockCloudflareClient.On("GetTunnelByName", ctx, ingressResourceName).Return(&cloudflare.Tunnel{ID: tunnelID, Name: ingressResourceName}, nil)
			mockCloudflareClient.On("UpsertTunnelConfiguration", ctx, tunnelID, mock.Anything).Return(nil)
			mockCloudflareClient.On("EnsureCNAMERecord", ctx, "zone-id-123", "karmada.example.com", fmt.Sprintf("%s.cfargotunnel.com", tunnelID)).Return("dns-record-id", nil)
			mockCloudflareClient.On("GetTunnelTokenByID", ctx, tunnelID).Return("test-token", nil)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ingressNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Mark deployment ready so the next reconcile proceeds to routing publication.
			connectorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-ingress-resource-connector", Namespace: "default"}, connectorDeployment)).To(Succeed())
			connectorDeployment.Status.ObservedGeneration = connectorDeployment.Generation
			connectorDeployment.Status.Replicas = 1
			connectorDeployment.Status.ReadyReplicas = 1
			connectorDeployment.Status.AvailableReplicas = 1
			Expect(k8sClient.Status().Update(ctx, connectorDeployment)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ingressNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			connectorConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-ingress-resource-connector-config", Namespace: "default"}, connectorConfigMap)).To(Succeed())
			Expect(connectorConfigMap.Data).To(HaveKey(connectorConfigFileName))
			Expect(connectorConfigMap.Data[connectorConfigFileName]).To(ContainSubstring("hostname: karmada.example.com"))
			Expect(connectorConfigMap.Data[connectorConfigFileName]).To(ContainSubstring("path: ^/api(/.*)?$"))
			Expect(connectorConfigMap.Data[connectorConfigFileName]).To(ContainSubstring("service: http://backend-svc.backend-ns.svc.cluster.local:8080"))
			Expect(connectorConfigMap.Data[connectorConfigFileName]).To(ContainSubstring("service: http_status:404"))

			connectorDeployment = &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-ingress-resource-connector", Namespace: "default"}, connectorDeployment)).To(Succeed())
			Expect(connectorDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(connectorDeployment.Spec.Template.Spec.Containers[0].Args).To(ContainElement("--config"))
			Expect(connectorDeployment.Spec.Template.Spec.Containers[0].Args).To(ContainElement("/etc/cloudflared/config/config.yaml"))
			Expect(connectorDeployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(HaveLen(1))
			Expect(connectorDeployment.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(connectorDeployment.Spec.Template.Spec.Volumes[0].ConfigMap).NotTo(BeNil())
			Expect(connectorDeployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name).To(Equal("test-ingress-resource-connector-config"))

			updatedTunnel := &networkingv1alpha1.CloudflareTunnel{}
			Expect(k8sClient.Get(ctx, ingressNamespacedName, updatedTunnel)).To(Succeed())
			Expect(updatedTunnel.Status.DNSHostname).To(Equal("karmada.example.com"))

			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should delete old dns record when hostname changes", func() {
			mockCloudflareClient := new(MockCloudflareClient)
			controllerReconciler := &CloudflareTunnelReconciler{}
			tunnel := &networkingv1alpha1.CloudflareTunnel{
				Spec: networkingv1alpha1.CloudflareTunnelSpec{
					Hostname: "karmada.20220625.xyz",
					ZoneID:   "zone-id-123",
					Ingress: &networkingv1alpha1.IngressSpec{
						Rules: []networkingv1alpha1.IngressRule{
							{
								Service: networkingv1alpha1.IngressServiceBackend{
									Name: "backend",
									Port: 8080,
								},
							},
						},
					},
				},
				Status: networkingv1alpha1.CloudflareTunnelStatus{
					DNSRecordID: "dns-record-old",
					DNSHostname: "k8s.20220625.xyz",
				},
			}

			mockCloudflareClient.
				On("DeleteDNSRecordByID", ctx, "zone-id-123", "dns-record-old").
				Return(nil).
				Once()
			mockCloudflareClient.
				On("EnsureCNAMERecord", ctx, "zone-id-123", "karmada.20220625.xyz", "test-tunnel-id.cfargotunnel.com").
				Return("dns-record-new", nil).
				Once()

			recordID, err := controllerReconciler.reconcileTunnelDNS(ctx, tunnel, mockCloudflareClient, "test-tunnel-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(recordID).To(Equal("dns-record-new"))
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
			resource.Status.TunnelID = tunnelID
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
			mockCloudflareClient.AssertNotCalled(GinkgoT(), "DeleteTunnelByID", mock.Anything, tunnelID)

			mockCloudflareClient.On("DeleteTunnelByID", ctx, tunnelID).Return(nil).Once()
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should delete managed dns record before deleting tunnel", func() {
			mockCloudflareClient := new(MockCloudflareClient)
			controllerReconciler := &CloudflareTunnelReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				CloudflareClient: mockCloudflareClient,
			}

			resource := &networkingv1alpha1.CloudflareTunnel{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			resource.Spec.ZoneID = "zone-id-123"
			resource.Finalizers = append(resource.Finalizers, finalizer)
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			resource.Status.TunnelID = tunnelID
			resource.Status.DNSRecordID = "dns-record-id"
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			mockCloudflareClient.On("DeleteDNSRecordByID", ctx, "zone-id-123", "dns-record-id").Return(nil).Once()
			mockCloudflareClient.On("DeleteTunnelByID", ctx, tunnelID).Return(nil).Once()

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should requeue when cloudflare tunnel delete returns retryable error", func() {
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
			resource.Status.TunnelID = tunnelID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			mockCloudflareClient.On("DeleteTunnelByID", ctx, tunnelID).Return(fmt.Errorf("conflict: tunnel has active connections")).Once()

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(deleteRetryAfter))

			resourceAfterFirstReconcile := &networkingv1alpha1.CloudflareTunnel{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resourceAfterFirstReconcile)).To(Succeed())
			Expect(resourceAfterFirstReconcile.Finalizers).To(ContainElement(finalizer))

			mockCloudflareClient.On("DeleteTunnelByID", ctx, tunnelID).Return(nil).Once()

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			resourceAfterSecondReconcile := &networkingv1alpha1.CloudflareTunnel{}
			err = k8sClient.Get(ctx, typeNamespacedName, resourceAfterSecondReconcile)
			if errors.IsNotFound(err) {
				mockCloudflareClient.AssertExpectations(GinkgoT())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(resourceAfterSecondReconcile.Finalizers).NotTo(ContainElement(finalizer))
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should treat not found as successful delete", func() {
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
			resource.Status.TunnelID = tunnelID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			mockCloudflareClient.On("DeleteTunnelByID", ctx, tunnelID).Return(fmt.Errorf("404 not found")).Once()

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			resourceAfterReconcile := &networkingv1alpha1.CloudflareTunnel{}
			err = k8sClient.Get(ctx, typeNamespacedName, resourceAfterReconcile)
			if errors.IsNotFound(err) {
				mockCloudflareClient.AssertExpectations(GinkgoT())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(resourceAfterReconcile.Finalizers).NotTo(ContainElement(finalizer))
			mockCloudflareClient.AssertExpectations(GinkgoT())
		})

		It("should skip cloudflare cleanup when credentials secret is missing during delete", func() {
			controllerReconciler := &CloudflareTunnelReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			resource := &networkingv1alpha1.CloudflareTunnel{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			resource.Finalizers = append(resource.Finalizers, finalizer)
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			resource.Status.TunnelID = tunnelID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			resourceAfterReconcile := &networkingv1alpha1.CloudflareTunnel{}
			err = k8sClient.Get(ctx, typeNamespacedName, resourceAfterReconcile)
			if errors.IsNotFound(err) {
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(resourceAfterReconcile.Finalizers).NotTo(ContainElement(finalizer))
		})
	})
})
