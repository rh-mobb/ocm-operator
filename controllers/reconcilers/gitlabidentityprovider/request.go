package gitlabidentityprovider

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
	"github.com/rh-mobb/ocm-operator/pkg/identityprovider"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

var (
	ErrMissingAccessToken  = errors.New("unable to locate gitlab api access token data")
	ErrMissingClientSecret = errors.New("unable to locate client secret data")
	ErrMissingCA           = errors.New("ca specified but unable to locate ca data")
)

// GitLabIdentityProviderRequest is an object that is unique to each reconciliation
// req.
type GitLabIdentityProviderRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.GitLabIdentityProvider
	Original          *ocmv1alpha1.GitLabIdentityProvider
	Desired           *ocmv1alpha1.GitLabIdentityProvider
	Trigger           triggers.Trigger
	Reconciler        *Controller
	GitLabClient      *identityprovider.GitLab
	OCMClient         *ocm.IdentityProviderClient

	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// data obtained during request reconciliation
	// AccessToken  string
	// ClientID     string
	ClientSecret string
	CA           string
}

func (r *Controller) NewRequest(ctx context.Context, ctrlReq ctrl.Request) (request.Request, error) {
	original := &ocmv1alpha1.GitLabIdentityProvider{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, ctrlReq.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &GitLabIdentityProviderRequest{}, err
	}

	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// get the client secret data from the cluster
	clientSecret, err := kubernetes.GetSecretData(
		ctx,
		r,
		original.Spec.ClientSecret.Name,
		ctrlReq.Namespace,
		ocmv1alpha1.GitLabClientSecretKey,
	)
	if clientSecret == "" {
		if err != nil {
			log.Log.Error(err, "error retrieving client secret")
		}

		return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to obtain client secret from cluster - %w", err)
	}

	// get the client secret data from the cluster
	ca, err := kubernetes.GetConfigMapData(ctx, r, original.Spec.CA.Name, ctrlReq.Namespace, ocmv1alpha1.GitLabCAKey)
	if clientSecret == "" {
		if err != nil {
			log.Log.Error(err, "error retrieving client secret")
		}

		return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to obtain client secret from cluster - %w", err)
	}

	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// // get the secret access token data from the cluster
	// accessToken, err := kubernetes.GetSecretData(ctx, r, original.Spec.AccessTokenSecret, req.Namespace, ocmv1alpha1.GitLabAccessTokenKey)
	// if accessToken == "" {
	// 	if err == nil {
	// 		return &GitLabIdentityProviderRequest{}, accessTokenError(original, ErrMissingAccessToken)
	// 	}

	// 	return &GitLabIdentityProviderRequest{}, accessTokenError(original, err)
	// }

	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// // create the api client used to interact with gitlab
	// gitlabClient, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(original.Spec.URL+"/api/v4"))
	// if err != nil {
	// 	return &GitLabIdentityProviderRequest{}, fmt.Errorf("error creating gitlab api client - %w", err)
	// }

	// create the desired state of the request based on the inputs
	desired := original.DeepCopy()
	if desired.Spec.DisplayName == "" {
		desired.Spec.DisplayName = desired.Name
	}

	return &GitLabIdentityProviderRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: ctrlReq,
		Context:           ctx,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
		// GitLabClient:      &identityprovider.GitLab{Client: gitlabClient},

		// data obtained from cluster
		// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
		// AccessToken: accessToken,
		ClientSecret: clientSecret,
		CA:           ca,
	}, nil
}

// DefaultRequeue returns the default requeue time for a request.
func (req *GitLabIdentityProviderRequest) DefaultRequeue() time.Duration {
	return defaultGitLabIdentityProviderRequeue
}

// GetObject returns the original object to satisfy the controllers.Request interface.
func (req *GitLabIdentityProviderRequest) GetObject() workload.Workload {
	return req.Original
}

// GetName returns the name as it should appear in OCM.
func (req *GitLabIdentityProviderRequest) GetName() string {
	return req.Desired.Spec.DisplayName
}

// GetClusterName returns the cluster name that this object belongs to.
func (req *GitLabIdentityProviderRequest) GetClusterName() string {
	return req.Desired.Spec.ClusterName
}

// GetContext returns the context of the request.
func (req *GitLabIdentityProviderRequest) GetContext() context.Context {
	return req.Context
}

// GetReconciler returns the context of the request.
func (req *GitLabIdentityProviderRequest) GetReconciler() kubernetes.Client {
	return req.Reconciler
}

// SetClusterStatus sets the relevant cluster fields in the status.  It is used
// to satisfy the request.Request interface.
func (req *GitLabIdentityProviderRequest) SetClusterStatus(cluster *clustersmgmtv1.Cluster) {
	if req.Original.Status.ClusterID == "" {
		req.Original.Status.ClusterID = cluster.ID()
	}

	if req.Original.Status.CallbackURL == "" {
		req.Original.Status.CallbackURL = ocm.GetCallbackURL(cluster, req.Desired.Spec.DisplayName)
	}
}

func (req *GitLabIdentityProviderRequest) desired() bool {
	if req.Desired == nil || req.Current == nil {
		return false
	}

	return reflect.DeepEqual(
		req.Desired.Spec,
		req.Current.Spec,
	)
}

// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
// func accessTokenError(from *ocmv1alpha1.GitLabIdentityProvider, err error) error {
// 	return fmt.Errorf(
// 		"unable to retrieve access token from [%s/%s] at key [%s] - %w",
// 		from.Namespace,
// 		from.Spec.AccessTokenSecret,
// 		ocmv1alpha1.GitLabAccessTokenKey,
// 		err,
// 	)
// }
