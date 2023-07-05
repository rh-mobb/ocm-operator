package ldapidentityprovider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

var (
	ErrMissingBindPassword = errors.New("unable to locate ldap bind password data")
	ErrMissingCA           = errors.New("ca specified but unable to locate ca data")
)

// LDAPIdentityProviderRequest is an object that is unique to each reconciliation
// req.
type LDAPIdentityProviderRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.LDAPIdentityProvider
	Original          *ocmv1alpha1.LDAPIdentityProvider
	Desired           *ocmv1alpha1.LDAPIdentityProvider
	Trigger           triggers.Trigger
	Reconciler        *Controller
	OCMClient         *ocm.IdentityProviderClient

	// data obtained during request reconciliation
	// TODO: Current* fields are not able to be pulled from OCM at this time as a security
	//       precaution.  Leaving them in place as a reminder but they are unused and
	//       ignored until we are provided a way to securely pull this data and compare it
	//       to the desired state.
	CurrentBindPassword string
	CurrentCA           string
	DesiredBindPassword string
	DesiredCA           string
}

// This controller must have the ability to pull secrets and configmaps which store the
// bind password and CA certificate data.

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *Controller) NewRequest(ctx context.Context, ctrlReq ctrl.Request) (request.Request, error) {
	original := &ocmv1alpha1.LDAPIdentityProvider{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, ctrlReq.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &LDAPIdentityProviderRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &LDAPIdentityProviderRequest{}, err
	}

	// get the bind password data from the cluster
	bindPassword, err := kubernetes.GetSecretData(ctx, r, original.Spec.BindPassword.Name, ctrlReq.Namespace, ocmv1alpha1.LDAPBindPasswordKey)
	if bindPassword == "" {
		if err != nil {
			log.Log.Error(err, "error retrieving bind password")
		}

		return &LDAPIdentityProviderRequest{}, bindPasswordError(original)
	}

	// get the ca config data from the cluster
	var ca string
	if original.Spec.CA.Name != "" {
		ca, err = kubernetes.GetConfigMapData(ctx, r, original.Spec.CA.Name, ctrlReq.Namespace, ocmv1alpha1.LDAPCAKey)
		if ca == "" {
			if err != nil {
				log.Log.Error(err, "error retrieving ca data")
			}

			return &LDAPIdentityProviderRequest{}, caCertError(original)
		}
	}

	// create the desired state of the request based on the inputs
	desired := original.DeepCopy()
	if desired.Spec.DisplayName == "" {
		desired.Spec.DisplayName = desired.Name
	}

	// ensure the attributes are defaulted
	desired.Spec.Attributes = ocmv1alpha1.LDAPAttributesToOpenShift(
		desired.Spec.Attributes.ID,
		desired.Spec.Attributes.Name,
		desired.Spec.Attributes.Email,
		desired.Spec.Attributes.PreferredUsername,
	)

	return &LDAPIdentityProviderRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: ctrlReq,
		Context:           ctx,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,

		// data obtained from cluster
		DesiredBindPassword: bindPassword,
		DesiredCA:           ca,
	}, nil
}

// DefaultRequeue returns the default requeue time for a request.
func (req *LDAPIdentityProviderRequest) DefaultRequeue() time.Duration {
	return defaultLDAPIdentityProviderRequeue
}

// GetObject returns the original object to satisfy the request.Request interface.
func (req *LDAPIdentityProviderRequest) GetObject() workload.Workload {
	return req.Original
}

// GetName returns the name as it should appear in OCM.
func (req *LDAPIdentityProviderRequest) GetName() string {
	return req.Desired.Spec.DisplayName
}

// GetClusterName returns the cluster name that this object belongs to.
func (req *LDAPIdentityProviderRequest) GetClusterName() string {
	return req.Desired.Spec.ClusterName
}

// GetContext returns the context of the request.
func (req *LDAPIdentityProviderRequest) GetContext() context.Context {
	return req.Context
}

// GetReconciler returns the context of the request.
func (req *LDAPIdentityProviderRequest) GetReconciler() kubernetes.Client {
	return req.Reconciler
}

// SetClusterStatus sets the relevant cluster fields in the status.  It is used
// to satisfy the request.Request interface.
func (req *LDAPIdentityProviderRequest) SetClusterStatus(cluster *clustersmgmtv1.Cluster) {
	if req.Original.Status.ClusterID == "" {
		req.Original.Status.ClusterID = cluster.ID()
	}
}

func (req *LDAPIdentityProviderRequest) desired() bool {
	if req.Desired == nil || req.Current == nil {
		return false
	}

	// TODO: leave this in place as a reminder however we cannot get the current password
	//        or CA data from the API likely due to security constraints, so we cannot
	//        compare them.  this means that the fields can be updated at create time
	//        but may not be updated (unless the object is changed by some other means)
	//
	// // ensure the passwords match
	// if req.DesiredBindPassword != req.CurrentBindPassword {
	// 	return false
	// }
	//
	// // ensure the ca data matches
	// if req.DesiredCA != req.CurrentCA {
	// 	return false
	// }

	return reflect.DeepEqual(
		req.Desired.Spec,
		req.Current.Spec,
	)
}

func bindPasswordError(from *ocmv1alpha1.LDAPIdentityProvider) error {
	return fmt.Errorf(
		"unable to retrieve bind password from secret [%s/%s] at key [%s] - %w",
		from.Namespace,
		from.Spec.BindPassword.Name,
		ocmv1alpha1.LDAPBindPasswordKey,
		ErrMissingBindPassword,
	)
}

func caCertError(from *ocmv1alpha1.LDAPIdentityProvider) error {
	return fmt.Errorf(
		"unable to retrieve ca cert from config map [%s/%s] at key [%s] - %w",
		from.Namespace,
		from.Spec.CA.Name,
		ocmv1alpha1.LDAPCAKey,
		ErrMissingCA,
	)
}
