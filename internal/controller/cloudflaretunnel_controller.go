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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/warjiang/cloudflare-tunnel-operator/api/v1alpha1"
	pkgcloudflare "github.com/warjiang/cloudflare-tunnel-operator/pkg/cloudflare"
)

const (
	// finalizer is the finalizer key for the CloudflareTunnel resource.
	finalizer = "networking.cloudflare-tunnel.spotty.com.cn/finalizer"
)

// CloudflareTunnelReconciler reconciles a CloudflareTunnel object
type CloudflareTunnelReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	CloudflareClient pkgcloudflare.ClientInterface
}

// +kubebuilder:rbac:groups=networking.cloudflare-tunnel.spotty.com.cn,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloudflare-tunnel.spotty.com.cn,resources=cloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloudflare-tunnel.spotty.com.cn,resources=cloudflaretunnels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CloudflareTunnel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *CloudflareTunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// Fetch the CloudflareTunnel instance
	tunnel := &networkingv1alpha1.CloudflareTunnel{}
	if err := r.Get(ctx, req.NamespacedName, tunnel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !tunnel.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, tunnel)
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(tunnel, finalizer) {
		controllerutil.AddFinalizer(tunnel, finalizer)
		if err := r.Update(ctx, tunnel); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile the tunnel
	return r.reconcile(ctx, tunnel)
}

func (r *CloudflareTunnelReconciler) reconcile(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if the tunnel exists
	cfTunnel, err := r.CloudflareClient.GetTunnelByName(ctx, tunnel.Spec.Name)
	if err != nil {
		// If the tunnel does not exist, create it
		if IsTunnelNotFoundError(err) {
			log.Info("creating cloudflare tunnel")
			newTunnel, secret, err := r.CloudflareClient.CreateTunnel(ctx, tunnel.Spec.Name)
			if err != nil {
				log.Error(err, "failed to create cloudflare tunnel")
				return ctrl.Result{}, err
			}
			cfTunnel = newTunnel

			// Create a secret to store the tunnel token
			if err := r.createTunnelSecret(ctx, tunnel, secret); err != nil {
				log.Error(err, "failed to create tunnel secret")
				return ctrl.Result{}, err
			}

		} else {
			log.Error(err, "failed to get cloudflare tunnel")
			return ctrl.Result{}, err
		}
	}

	// Update status
	tunnel.Status.TunnelID = cfTunnel.ID
	if err := r.Status().Update(ctx, tunnel); err != nil {
		log.Error(err, "failed to update tunnel status")
		return ctrl.Result{}, err
	}

	// Check if the secret exists
	secretName := fmt.Sprintf("%s-token", tunnel.Name)
	var secret corev1.Secret
	err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: tunnel.Namespace}, &secret)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("tunnel secret not found, creating it")
		// If the secret does not exist, create it
		token, err := r.CloudflareClient.GetTunnelTokenByID(ctx, cfTunnel.ID)
		if err != nil {
			log.Error(err, "failed to get tunnel token")
			return ctrl.Result{}, err
		}
		if err := r.createTunnelSecret(ctx, tunnel, []byte(token)); err != nil {
			log.Error(err, "failed to create tunnel secret")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get tunnel secret")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CloudflareTunnelReconciler) reconcileDelete(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if controllerutil.ContainsFinalizer(tunnel, finalizer) {
		// Delete the tunnel
		if tunnel.Status.TunnelID != "" {
			log.Info("deleting cloudflare tunnel")
			if err := r.CloudflareClient.DeleteTunnelByID(ctx, tunnel.Status.TunnelID); err != nil {
				// If the tunnel is already deleted, we can ignore the error
				if !IsTunnelNotFoundError(err) {
					log.Error(err, "failed to delete cloudflare tunnel")
					return ctrl.Result{}, err
				}
			}
		}

		// Remove the finalizer
		controllerutil.RemoveFinalizer(tunnel, finalizer)
		if err := r.Update(ctx, tunnel); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// IsTunnelNotFoundError checks if the error is a tunnel not found error.
func IsTunnelNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func (r *CloudflareTunnelReconciler) createTunnelSecret(ctx context.Context, tunnel *networkingv1alpha1.CloudflareTunnel, secret []byte) error {
	secretName := fmt.Sprintf("%s-token", tunnel.Name)
	secretData := map[string][]byte{
		"token": secret,
	}

	// Create the secret
	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: tunnel.Namespace,
		},
		Data: secretData,
	}

	// Set the owner reference
	if err := controllerutil.SetControllerReference(tunnel, secretObj, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, secretObj)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareTunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CloudflareTunnel{}).
		Named("cloudflaretunnel").
		Complete(r)
}
