package phases

import (
	"errors"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/internal/factory"
	ctrl "sigs.k8s.io/controller-runtime"
)

type testClusterFetcher struct {
	cluster *clustersmgmtv1.Cluster
	err     error
}

func newTestClusterFetcher(state clustersmgmtv1.ClusterState, err bool) *testClusterFetcher {
	fetcher := &testClusterFetcher{}

	if state != "" {
		builder := &clustersmgmtv1.ClusterBuilder{}

		builder.State(state).ID(factory.DefaultClusterID)

		cluster, _ := builder.Build()

		fetcher.cluster = cluster
	}

	if err {
		fetcher.err = errors.New("cluster fetcher error")
	}

	return fetcher
}

func (t *testClusterFetcher) Get() (*clustersmgmtv1.Cluster, error) { return t.cluster, t.err }
func (t *testClusterFetcher) For(s string) *clustersmgmtv1.ClusterClient {
	return &clustersmgmtv1.ClusterClient{}
}

func TestHandleClusterPhase(t *testing.T) {
	requeueResult, _ := requeue.After(defaultMissingUpstreamRequeue, nil)

	t.Parallel()

	type args struct {
		req     request.Cluster
		client  request.ClusterFetcher
		trigger triggers.Trigger
		logger  logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "ensure failing to get upstream cluster fails",
			args: args{
				req:     factory.NewTestRequest(defaultMissingUpstreamRequeue, factory.NewTestWorkload("")),
				client:  newTestClusterFetcher(clustersmgmtv1.ClusterStateError, true),
				trigger: triggers.Create,
				logger:  ctrl.Log.WithName("upstream-cluster-fail"),
			},
			want:    requeueResult,
			wantErr: true,
		},
		{
			name: "ensure failing to set condition fails",
			args: args{
				req:     factory.NewTestErrorRequest(defaultMissingUpstreamRequeue, factory.NewTestWorkload("")),
				client:  newTestClusterFetcher(clustersmgmtv1.ClusterStateReady, false),
				trigger: triggers.Create,
				logger:  ctrl.Log.WithName("set-condition-fail"),
			},
			want:    requeueResult,
			wantErr: true,
		},
		{
			name: "ensure requeue without error on missing cluster",
			args: args{
				req:     factory.NewTestRequest(defaultMissingUpstreamRequeue, factory.NewTestWorkload("")),
				client:  newTestClusterFetcher("", false),
				trigger: triggers.Create,
				logger:  ctrl.Log.WithName("requeue-on-missing"),
			},
			want:    requeueResult,
			wantErr: false,
		},
		{
			name: "ensure requeue without error on non-ready cluster",
			args: args{
				req:     factory.NewTestRequest(defaultMissingUpstreamRequeue, factory.NewTestWorkload("")),
				client:  newTestClusterFetcher(clustersmgmtv1.ClusterStateInstalling, false),
				trigger: triggers.Create,
				logger:  ctrl.Log.WithName("requeue-on-non-ready"),
			},
			want:    requeueResult,
			wantErr: false,
		},
		{
			name: "ensure no requeue without error on ready cluster",
			args: args{
				req:     factory.NewTestRequest(defaultMissingUpstreamRequeue, factory.NewTestWorkload("")),
				client:  newTestClusterFetcher(clustersmgmtv1.ClusterStateReady, false),
				trigger: triggers.Create,
				logger:  ctrl.Log.WithName("no-requeue-ready"),
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := HandleClusterPhase(tt.args.req, tt.args.client, tt.args.trigger, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleClusterPhase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleClusterPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
