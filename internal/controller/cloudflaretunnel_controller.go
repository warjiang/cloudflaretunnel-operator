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

	credentialsSecretKeyAPIToken  = "api-token"
	credentialsSecretKeyAccountID = "account-id"
	tunnelTokenSecretKey          = "token"
)

// CloudflareTunnelReconciler reconciles a CloudflareTunnel object
type CloudflareTunnelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// CloudflareClient can be injected by tests. In production this should be nil,
	// and the reconciler builds a client from credentialsRef on each reconcile.
	CloudflareClient pkgcloudflare.ClientInterface
}

// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflaretunnel.spotty.com.cn,resources=cloudflaretunnels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
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
		r.setCondition(tunnel, conditionReady, metav1.ConditionFalse, "SecretSyncFailed", err.Error())
		_ = r.updateStatus(ctx, tunnel)
		return ctrl.Result{}, err
	}

	r.setCondition(tunnel, conditionTokenSecretReady, metav1.ConditionTrue, "SecretSynced", "Tunnel token secret is synced")
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
		return ctrl.Result{}, deleteErr
	}

	controllerutil.RemoveFinalizer(tunnel, finalizer)
	if err := r.Update(ctx, tunnel); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
