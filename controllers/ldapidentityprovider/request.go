package ldapidentityprovider

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

var (
	ErrMissingClusterID                   = errors.New("unable to find cluster id")
	ErrMissingBindPassword                = errors.New("unable to locate ldap bind password data")
	ErrMissingCA                          = errors.New("ca specified but unable to locate ca data")
	ErrLDAPIdentityProviderRequestConvert = errors.New("unable to convert generic request to ldap identity provider request")
)

// LDAPIdentityProviderRequest is an object that is unique to each reconciliation
// request.
type LDAPIdentityProviderRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.LDAPIdentityProvider
	Original          *ocmv1alpha1.LDAPIdentityProvider
	Desired           *ocmv1alpha1.LDAPIdentityProvider
	Log               logr.Logger
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

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.LDAPIdentityProvider{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &LDAPIdentityProviderRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &LDAPIdentityProviderRequest{}, err
	}

	// get the bind password data from the cluster
	bindPassword, err := kubernetes.GetSecretData(ctx, r, original.Spec.BindPassword.Name, req.Namespace, ocmv1alpha1.LDAPBindPasswordKey)
	if bindPassword == "" {
		if err != nil {
			log.Log.Error(err, "error retrieving bind password")
		}

		return &LDAPIdentityProviderRequest{}, bindPasswordError(original)
	}

	// get the ca config data from the cluster
	var ca string
	if original.Spec.CA.Name != "" {
		ca, err = kubernetes.GetConfigMapData(ctx, r, original.Spec.CA.Name, req.Namespace, ocmv1alpha1.LDAPCAKey)
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
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,

		// data obtained from cluster
		DesiredBindPassword: bindPassword,
		DesiredCA:           ca,
	}, nil
}

// GetObject returns the original object to satisfy the controllers.Request interface.
func (request *LDAPIdentityProviderRequest) GetObject() workload.Workload {
	return request.Original
}

// GetName returns the name as it should appear in OCM.
func (request *LDAPIdentityProviderRequest) GetName() string {
	return request.Desired.Spec.DisplayName
}

// updateStatusCluster updates fields related to the cluster in which the machine pool resides in.
func (request *LDAPIdentityProviderRequest) updateStatusCluster() error {
	// retrieve the cluster id
	clusterClient := ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.ClusterName)
	cluster, err := clusterClient.Get()
	if err != nil || cluster == nil {
		return fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// if the cluster id is missing return an error
	if cluster.ID() == "" {
		return fmt.Errorf("missing cluster id in response - %w", ErrMissingClusterID)
	}

	// keep track of the original object
	original := request.Original.DeepCopy()
	request.Original.Status.ClusterID = cluster.ID()

	// store the cluster id in the status
	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf(
			"unable to update status.clusterID=%s - %w",
			cluster.ID(),
			err,
		)
	}

	return nil
}

func (request *LDAPIdentityProviderRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	// TODO: leave this in place as a reminder however we cannot get the current password
	//        or CA data from the API likely due to security constraints, so we cannot
	//        compare them.  this means that the fields can be updated at create time
	//        but may not be updated (unless the object is changed by some other means)
	//
	// // ensure the passwords match
	// if request.DesiredBindPassword != request.CurrentBindPassword {
	// 	return false
	// }
	//
	// // ensure the ca data matches
	// if request.DesiredCA != request.CurrentCA {
	// 	return false
	// }

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
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
