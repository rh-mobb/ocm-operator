package controllers

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
)

func TestMachinePoolRequest_desired(t *testing.T) {
	t.Parallel()

	object := &ocmv1alpha1.MachinePool{
		Spec: ocmv1alpha1.MachinePoolSpec{
			DisplayName:         "test",
			ClusterName:         "test",
			MinimumNodesPerZone: 1,
			MaximumNodesPerZone: 1,
			InstanceType:        "m5.xlarge",
			Labels: map[string]string{
				"this": "that",
			},
			Taints: []corev1.Taint{
				{
					Key:       "this",
					Value:     "that",
					Effect:    corev1.TaintEffectNoSchedule,
					TimeAdded: nil,
				},
			},
			AWS: ocmv1alpha1.MachinePoolProviderAWS{
				SpotInstances: ocmv1alpha1.MachinePoolProviderAWSSpotInstances{
					Enabled:      false,
					MaximumPrice: 0,
				},
			},
		},
	}

	type fields struct {
		Current *ocmv1alpha1.MachinePool
		Desired *ocmv1alpha1.MachinePool
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "ensure equal objects reflect desired state",
			fields: fields{
				Current: object.DeepCopy(),
				Desired: object.DeepCopy(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request := &MachinePoolRequest{
				Current:  tt.fields.Current,
				Original: tt.fields.Desired,
			}
			if got := request.desired(); got != tt.want {
				t.Errorf("MachinePoolRequest.desired() = %v, want %v", got, tt.want)
			}
		})
	}
}
