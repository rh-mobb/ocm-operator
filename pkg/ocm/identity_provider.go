package ocm

import (
	"fmt"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	callbackURLPrefix = "oauth"
)

func GetCallbackURL(cluster *clustersmgmtv1.Cluster, name string) string {
	return fmt.Sprintf("%s.%s.%s/oauth2callback/%s",
		callbackURLPrefix,
		cluster.Name(),
		cluster.DNS().BaseDomain(),
		name,
	)
}
