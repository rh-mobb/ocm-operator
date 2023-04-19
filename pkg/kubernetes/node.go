package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nodeConditionReady = "Ready"
)

func GetLabeledNodes(ctx context.Context, c Client, nodeLabels map[string]string) (*corev1.NodeList, error) {
	nodeList := corev1.NodeList{}

	// list the nodes that have the appropriate labels. this ensures that we only find
	// nodes with the proper labels to include our own managed labels
	if err := c.List(
		ctx,
		&nodeList,
		&client.ListOptions{LabelSelector: labels.SelectorFromSet(nodeLabels)},
	); err != nil {
		return &nodeList, fmt.Errorf("error listing nodes - %w", err)
	}

	return &nodeList, nil
}

func NodesAreReady(nodes ...corev1.Node) bool {
	for _, node := range nodes {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeConditionType(nodeConditionReady) && condition.Status == corev1.ConditionFalse {
				return false
			}
		}
	}

	return true
}
