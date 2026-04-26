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
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/warjiang/cloudflaretunnel-operator/api/v1alpha1"
	pkgcloudflare "github.com/warjiang/cloudflaretunnel-operator/pkg/cloudflare"
)

const (
	finalizer = "cloudflaretunnel.spotty.com.cn/finalizer"

	conditionReady            = "Ready"
	conditionCredentialsReady = "CredentialsReady"
	conditionTunnelReady      = "TunnelReady"
	conditionTokenSecretReady = "TokenSecretReady"
	conditionConnectorReady   = "ConnectorReady"

	credentialsSecretKeyAPIToken  = "api-token"
	credentialsSecretKeyAccountID = "account-id"
	tunnelTokenSecretKey          = "token"

	defaultConnectorImage    = "cloudflare/cloudflared:2026.3.0"
	defaultConnectorReplicas = int32(1)
	deleteRetryAfter         = 10 * time.Second
)

// CloudflareTunnelReconciler reconciles a CloudflareTunnel object
type CloudflareTunnelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// CloudflareClient can be injected by tests. In production this should be nil,
	// and the reconciler builds a client from credentialsRef on each reconcile.
	CloudflareClient pkgcloudflare.ClientInterface

	// Connector defaults used when spec.connector fields are omitted.
	ConnectorDefaultImage    string
	ConnectorDefaultReplicas int32
}

// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
func (r *CloudflareTunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	tunnel := &networkingv1alpha1.CloudflareTunnel{}
	if err := r.Get(ctx, req.NamespacedName, tunnel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !tunnel.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, tunnel)
	}

	if !controllerutil.ContainsFinalizer(tunnel, finalizer) {
		controllerutil.AddFinalizer(tunnel, finalizer)
		if err := r.Update(ctx, tunnel); err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.reconcile(ctx, tunnel)
}

func (r *CloudflareTunnelReconciler) reconcile(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	cfClient, err := r.getCloudflareClient(ctx, tunnel)
	if err != nil {
		r.setCondition(tunnel, conditionCredentialsReady, metav1.ConditionFalse, "InvalidCredentials", err.Error())
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "InvalidCredentials", err.Error())
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	r.setCondition(tunnel, conditionCredentialsReady, metav1.ConditionTrue, "CredentialsLoaded", "Credentials loaded successfully")

	cfTunnel, err := cfClient.GetTunnelByName(ctx, tunnel.Spec.Name)
	if err != nil {
		if IsTunnelNotFoundError(err) {
			log.Info("Creating Cloudflare tunnel", "name", tunnel.Spec.Name)
			newTunnel, _, createErr := cfClient.CreateTunnel(ctx, tunnel.Spec.Name)
			if createErr != nil {
				r.setCondition(tunnel, conditionTunnelReady, metav1.ConditionFalse, "CreateFailed", createErr.Error())
				r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "CreateFailed", createErr.Error())
				_ = r.updateStatus(ctx, tunnel)
				return ctrl.Result{}, createErr
			}
			cfTunnel = newTunnel
		} else {
			r.setCondition(tunnel, conditionTunnelReady, metav1.ConditionFalse, "LookupFailed", err.Error())
			r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "LookupFailed", err.Error())
			_ = r.updateStatus(ctx, tunnel)
			return ctrl.Result{}, err
		}
	}

	tunnel.Status.TunnelID = cfTunnel.ID
	r.setCondition(tunnel, conditionTunnelReady, metav1.ConditionTrue, "TunnelReady", "Tunnel exists")

	token, err := cfClient.GetTunnelTokenByID(ctx, cfTunnel.ID)
	if err != nil {
		r.setCondition(tunnel, conditionTokenSecretReady, metav1.ConditionFalse, "TokenFetchFailed", err.Error())
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "TokenFetchFailed", err.Error())
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{}, err
	}

	if err := r.reconcileTokenSecret(ctx, tunnel, token); err != nil {
		r.setCondition(tunnel, conditionTokenSecretReady, metav1.ConditionFalse, "SecretSyncFailed", err.Error())
		r.setCondition(tunnel, conditionConnectorReady, metav1.ConditionFalse, "Pending", "Token secret is not ready")
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "SecretSyncFailed", err.Error())
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{}, err
	}

	r.setCondition(tunnel, conditionTokenSecretReady, metav1.ConditionTrue, "SecretSynced", "Tunnel token secret is synced")

	connectorDeployment, err := r.reconcileConnectorDeployment(ctx, tunnel)
	if err != nil {
		r.setCondition(tunnel, conditionConnectorReady, metav1.ConditionFalse, "DeploymentSyncFailed", err.Error())
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "DeploymentSyncFailed", err.Error())
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{}, err
	}

	if !isConnectorDeploymentReady(connectorDeployment) {
		msg := fmt.Sprintf(
			"Waiting for connector deployment: availableReplicas=%d, desiredReplicas=%d",
			connectorDeployment.Status.AvailableReplicas,
			connectorDeployment.Status.Replicas,
		)
		r.setCondition(tunnel, conditionConnectorReady, metav1.ConditionFalse, "DeploymentNotReady", msg)
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "DeploymentNotReady", msg)
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	r.setCondition(tunnel, conditionConnectorReady, metav1.ConditionTrue, "DeploymentReady", "Connector deployment is ready")
	r.setCondition(tunnel, conditionReady, metav1.ConditionTrue, "Ready", "Cloudflare tunnel is ready")

	if err := r.updateStatus(ctx, tunnel); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CloudflareTunnelReconciler) reconcileDelete(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(tunnel, finalizer) {
		return ctrl.Result{}, nil
	}

	connectorStopped, err := r.ensureConnectorStoppedForDelete(ctx, tunnel)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !connectorStopped {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	cfClient, err := r.getCloudflareClient(ctx, tunnel)
	if err != nil {
		return ctrl.Result{}, err
	}

	deleteErr := error(nil)
	if tunnel.Status.TunnelID != "" {
		log.Info("Deleting Cloudflare tunnel by status ID", "tunnelID", tunnel.Status.TunnelID)
		deleteErr = cfClient.DeleteTunnelByID(ctx, tunnel.Status.TunnelID)
	} else {
		log.Info("Deleting Cloudflare tunnel by name", "name", tunnel.Spec.Name)
		deleteErr = cfClient.DeleteTunnelByName(ctx, tunnel.Spec.Name)
	}
	if deleteErr != nil && !IsTunnelNotFoundError(deleteErr) {
		if IsTunnelDeleteRetryableError(deleteErr) {
			log.Info("Cloudflare tunnel delete returned retryable error, will retry", "error", deleteErr.Error(), "requeueAfter", deleteRetryAfter)
			return ctrl.Result{RequeueAfter: deleteRetryAfter}, nil
		}
		return ctrl.Result{}, deleteErr
	}

	controllerutil.RemoveFinalizer(tunnel, finalizer)
	if err := r.Update(ctx, tunnel); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CloudflareTunnelReconciler) ensureConnectorStoppedForDelete(
	ctx context.Context,
	tunnel *networkingv1alpha1.CloudflareTunnel,
) (bool, error) {
	log := logf.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	deploymentKey := client.ObjectKey{
		Name:      desiredConnectorDeploymentName(tunnel),
		Namespace: tunnel.Namespace,
	}
	if err := r.Get(ctx, deploymentKey, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
	} else {
		if deployment.DeletionTimestamp.IsZero() {
			log.Info("Deleting connector deployment before tunnel cleanup", "deployment", deployment.Name)
			if err := r.Delete(ctx, deployment); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
		// The deployment existed in this reconcile loop. Requeue to give the
		// workload time to stop before deleting the Cloudflare tunnel.
		return false, nil
	}

	labels := connectorLabels(tunnel)
	if err := r.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(tunnel.Namespace), client.MatchingLabels(labels)); err != nil {
		return false, err
	}

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(tunnel.Namespace), client.MatchingLabels(labels)); err != nil {
		return false, err
	}
	if len(podList.Items) > 0 {
		log.Info("Waiting for connector pods to terminate before tunnel cleanup", "remainingPods", len(podList.Items))
		return false, nil
	}

	return true, nil
}

func (r *CloudflareTunnelReconciler) getCloudflareClient(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) (pkgcloudflare.ClientInterface, error) {
	if r.CloudflareClient != nil {
		return r.CloudflareClient, nil
	}

	credentialsRefName := tunnel.Spec.CredentialsRef.Name
	if credentialsRefName == "" {
		return nil, fmt.Errorf("spec.credentialsRef.name is required")
	}

	credentialsSecret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: credentialsRefName, Namespace: tunnel.Namespace}, credentialsSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("credentials secret %q not found", credentialsRefName)
		}
		return nil, err
	}

	apiToken := string(credentialsSecret.Data[credentialsSecretKeyAPIToken])
	accountID := string(credentialsSecret.Data[credentialsSecretKeyAccountID])
	if apiToken == "" {
		return nil, fmt.Errorf("credentials secret %q missing key %q", credentialsRefName, credentialsSecretKeyAPIToken)
	}
	if accountID == "" {
		return nil, fmt.Errorf("credentials secret %q missing key %q", credentialsRefName, credentialsSecretKeyAccountID)
	}

	return pkgcloudflare.NewClient(accountID,
		pkgcloudflare.WithAccountID(accountID),
		pkgcloudflare.WithAPIToken(apiToken),
	)
}

func (r *CloudflareTunnelReconciler) reconcileTokenSecret(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel, token string) error {
	secretName := r.desiredTokenSecretName(tunnel)
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: tunnel.Namespace}}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[tunnelTokenSecretKey] = []byte(token)
		secret.Type = corev1.SecretTypeOpaque
		return controllerutil.SetControllerReference(tunnel, secret, r.Scheme)
	})
	return err
}

func (r *CloudflareTunnelReconciler) desiredTokenSecretName(tunnel *networkingv1alpha1.CloudflareTunnel) string {
	if tunnel.Spec.TokenSecretRef != nil && tunnel.Spec.TokenSecretRef.Name != "" {
		return tunnel.Spec.TokenSecretRef.Name
	}
	return fmt.Sprintf("%s-token", tunnel.Name)
}

func (r *CloudflareTunnelReconciler) reconcileConnectorDeployment(
	ctx context.Context,
	tunnel *networkingv1alpha1.CloudflareTunnel,
) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desiredConnectorDeploymentName(tunnel),
			Namespace: tunnel.Namespace,
		},
	}

	image := r.ConnectorDefaultImage
	if image == "" {
		image = defaultConnectorImage
	}

	replicas := r.ConnectorDefaultReplicas
	if replicas <= 0 {
		replicas = defaultConnectorReplicas
	}

	var resources corev1.ResourceRequirements
	var nodeSelector map[string]string
	var tolerations []corev1.Toleration
	if tunnel.Spec.Connector != nil {
		if tunnel.Spec.Connector.Image != "" {
			image = tunnel.Spec.Connector.Image
		}
		if tunnel.Spec.Connector.Replicas != nil && *tunnel.Spec.Connector.Replicas > 0 {
			replicas = *tunnel.Spec.Connector.Replicas
		}
		resources = tunnel.Spec.Connector.Resources
		nodeSelector = tunnel.Spec.Connector.NodeSelector
		tolerations = tunnel.Spec.Connector.Tolerations
	}

	labels := connectorLabels(tunnel)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = labels
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: labels,
		}
		deployment.Spec.Template.Labels = labels
		deployment.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:            "cloudflared",
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args: []string{
					"tunnel",
					"--no-autoupdate",
					"run",
					"--token",
					"$(TUNNEL_TOKEN)",
				},
				Env: []corev1.EnvVar{
					{
						Name: "TUNNEL_TOKEN",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: r.desiredTokenSecretName(tunnel)},
								Key:                  tunnelTokenSecretKey,
							},
						},
					},
				},
				Resources: resources,
			},
		}
		deployment.Spec.Template.Spec.NodeSelector = nodeSelector
		deployment.Spec.Template.Spec.Tolerations = tolerations
		return controllerutil.SetControllerReference(tunnel, deployment, r.Scheme)
	})
	if err != nil {
		return nil, err
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func desiredConnectorDeploymentName(tunnel *networkingv1alpha1.CloudflareTunnel) string {
	name := fmt.Sprintf("%s-connector", tunnel.Name)
	if len(name) <= 63 {
		return name
	}
	return strings.TrimSuffix(name[:63], "-")
}

func connectorLabels(tunnel *networkingv1alpha1.CloudflareTunnel) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "cloudflared",
		"app.kubernetes.io/managed-by": "cloudflaretunnel-operator",
		"app.kubernetes.io/instance":   tunnel.Name,
		"cloudflaretunnel/name":        tunnel.Name,
	}
}

func isConnectorDeploymentReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.AvailableReplicas > 0
}

func (r *CloudflareTunnelReconciler) setCondition(
	tunnel *networkingv1alpha1.CloudflareTunnel,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) {
	apimeta.SetStatusCondition(&tunnel.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: tunnel.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *CloudflareTunnelReconciler) updateStatus(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) error {
	tunnel.Status.ObservedGeneration = tunnel.Generation
	return r.Status().Update(ctx, tunnel)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareTunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CloudflareTunnel{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Named("cloudflaretunnel").
		Complete(r)
}

// IsTunnelNotFoundError checks if the error is a tunnel not found error.
func IsTunnelNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

// IsTunnelDeleteRetryableError checks whether a delete failure is transient and should be retried.
func IsTunnelDeleteRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"active connection",
		"active connections",
		"has connections",
		"in use",
		"conflict",
		"409",
		"temporarily unavailable",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
