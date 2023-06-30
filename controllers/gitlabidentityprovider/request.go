package gitlabidentityprovider

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/identityprovider"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

var (
	ErrMissingClusterID                     = errors.New("unable to find cluster id")
	ErrMissingAccessToken                   = errors.New("unable to locate gitlab api access token data")
	ErrMissingClientSecret                  = errors.New("unable to locate client secret data")
	ErrMissingCA                            = errors.New("ca specified but unable to locate ca data")
	ErrGitLabIdentityProviderRequestConvert = errors.New("unable to convert generic request to gitlab identity provider request")
)

// GitLabIdentityProviderRequest is an object that is unique to each reconciliation
// request.
type GitLabIdentityProviderRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.GitLabIdentityProvider
	Original          *ocmv1alpha1.GitLabIdentityProvider
	Desired           *ocmv1alpha1.GitLabIdentityProvider
	Log               logr.Logger
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

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.GitLabIdentityProvider{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &GitLabIdentityProviderRequest{}, err
	}

	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// get the client secret data from the cluster
	clientSecret, err := kubernetes.GetSecretData(ctx, r, original.Spec.ClientSecret.Name, req.Namespace, ocmv1alpha1.GitLabClientSecretKey)
	if clientSecret == "" {
		if err != nil {
			log.Log.Error(err, "error retrieving client secret")
		}

		return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to obtain client secret from cluster - %w", err)
	}

	// get the client secret data from the cluster
	ca, err := kubernetes.GetConfigMapData(ctx, r, original.Spec.CA.Name, req.Namespace, ocmv1alpha1.GitLabCAKey)
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
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
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

// GetObject returns the original object to satisfy the controllers.Request interface.
func (request *GitLabIdentityProviderRequest) GetObject() workload.Workload {
	return request.Original
}

// GetName returns the name as it should appear in OCM.
func (request *GitLabIdentityProviderRequest) GetName() string {
	return request.Desired.Spec.DisplayName
}

// GetClusterName returns the cluster name that this object belongs to.
func (request *GitLabIdentityProviderRequest) GetClusterName() string {
	return request.Desired.Spec.ClusterName
}

// GetContext returns the context of the request.
func (request *GitLabIdentityProviderRequest) GetContext() context.Context {
	return request.Context
}

// GetReconciler returns the context of the request.
func (request *GitLabIdentityProviderRequest) GetReconciler() kubernetes.Client {
	return request.Reconciler
}

// SetClusterStatus sets the relevant cluster fields in the status.  It is used
// to satisfy the controllers.Request interface.
func (request *GitLabIdentityProviderRequest) SetClusterStatus(cluster *clustersmgmtv1.Cluster) {
	if request.Original.Status.ClusterID == "" {
		request.Original.Status.ClusterID = cluster.ID()
	}

	if request.Original.Status.CallbackURL == "" {
		request.Original.Status.CallbackURL = ocm.GetCallbackURL(cluster, request.Desired.Spec.DisplayName)
	}
}

func (request *GitLabIdentityProviderRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
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
