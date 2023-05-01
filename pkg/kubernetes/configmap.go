package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetConfigMapData(ctx context.Context, c Client, name, namespace, key string) (string, error) {
	configMap := &corev1.ConfigMap{}

	if err := c.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, configMap); err != nil {
		return "", fmt.Errorf(
			"unable to retrieve configmap [%s/%s] from cluster - %w",
			namespace,
			name,
			err,
		)
	}

	if configMap.Data == nil || len(configMap.Data[key]) == 0 {
		return "", nil
	}

	return string(configMap.Data[key]), nil
}
