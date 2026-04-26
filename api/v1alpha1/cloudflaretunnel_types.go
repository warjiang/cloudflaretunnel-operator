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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudflareTunnelSpec defines the desired state of CloudflareTunnel.
// +kubebuilder:validation:XValidation:rule="has(self.tunnelName) || has(self.name)",message="spec.tunnelName is required (legacy spec.name is still supported)"
type CloudflareTunnelSpec struct {
	// TunnelName is the Cloudflare-side tunnel name used to create/query the tunnel.
	// It is independent from Kubernetes metadata.name and does not need to match it.
	// +optional
	TunnelName string `json:"tunnelName,omitempty"`

	// Name is deprecated, use tunnelName instead.
	// Kept for backward compatibility with older manifests.
	// +optional
	Name string `json:"name,omitempty"`

	// CredentialsRef points to a Secret that stores Cloudflare credentials.
	// Required keys:
	// - api-token
	// - account-id
	// +kubebuilder:validation:Required
	CredentialsRef CredentialsSecretRef `json:"credentialsRef"`

	// TokenSecretRef is where the tunnel token will be stored.
	// If omitted, "<metadata.name>-token" is used.
	// +optional
	TokenSecretRef *TokenSecretRef `json:"tokenSecretRef,omitempty"`

	// Connector defines how the cloudflared workload is run in Kubernetes.
	// +optional
	Connector *ConnectorSpec `json:"connector,omitempty"`
}

// ConnectorSpec defines the desired cloudflared workload configuration.
type ConnectorSpec struct {
	// Image is the cloudflared container image.
	// If omitted, controller default image is used.
	// +optional
	Image string `json:"image,omitempty"`

	// Replicas is the desired number of cloudflared pods for this tunnel.
	// If omitted, controller default replicas is used.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources describes compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations are attached to the cloudflared pod.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// CredentialsSecretRef references the Secret containing Cloudflare credentials.
type CredentialsSecretRef struct {
	// Name is the Secret name in the same namespace as the CloudflareTunnel.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// TokenSecretRef references the Secret where the tunnel token is stored.
type TokenSecretRef struct {
	// Name is the Secret name in the same namespace as the CloudflareTunnel.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// CloudflareTunnelStatus defines the observed state of CloudflareTunnel.
type CloudflareTunnelStatus struct {
	// TunnelID is the unique identifier of the Cloudflare tunnel.
	// +optional
	TunnelID string `json:"tunnelID,omitempty"`

	// Conditions represent the latest available observations of a CloudflareTunnel's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CloudflareTunnel is the Schema for the cloudflaretunnels API.
type CloudflareTunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareTunnelSpec   `json:"spec,omitempty"`
	Status CloudflareTunnelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareTunnelList contains a list of CloudflareTunnel.
type CloudflareTunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudflareTunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudflareTunnel{}, &CloudflareTunnelList{})
}
