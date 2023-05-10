package ocm

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	ver "github.com/hashicorp/go-version"
	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrVersionNotFound  = errors.New("unable to find version")
	ErrVersionsNotFound = errors.New("unable to find available versions")
)

// GetVersionObject returns the version object for a particular raw version.  It assumes stable channel only
// as well as only versions that are enabled and available in ROSA.
func GetVersionObject(connection *sdk.Connection, rawVersion string) (version *clustersmgmtv1.Version, err error) {
	filter := fmt.Sprintf("enabled = 'true' AND rosa_enabled = 'true' AND raw_id = '%s' AND channel_group = 'stable'", rawVersion)

	versions, err := connection.ClustersMgmt().V1().Versions().List().Search(filter).Send()
	if err != nil {
		return version, fmt.Errorf("unable to get versions - %w", err)
	}

	if len(versions.Items().Slice()) == 0 {
		return version, ErrVersionNotFound
	}

	// sort list in descending order
	sort.Slice(versions.Items().Slice(), func(i, j int) bool {
		a, erra := ver.NewVersion(versions.Items().Slice()[i].RawID())
		b, errb := ver.NewVersion(versions.Items().Slice()[j].RawID())
		if erra != nil || errb != nil {
			return false
		}
		return a.GreaterThan(b)
	})

	return versions.Items().Slice()[0], nil
}

// MajorMinorVersion returns the major and minor representation of a version object.
func MajorMinorVersion(version *clustersmgmtv1.Version) string {
	versionSplit := strings.Split(version.RawID(), ".")

	return fmt.Sprintf("%s.%s", versionSplit[0], versionSplit[1])
}

// GetAvailableVersions gets all available versions from OCM.
// Copied from https://github.com/openshift/rosa/blob/master/pkg/ocm/versions.go#L54
func GetAvailableVersions(connection *sdk.Connection) (versions []*clustersmgmtv1.Version, err error) {
	collection := connection.ClustersMgmt().V1().Versions()
	page := 1
	size := 100
	filter := "enabled = 'true' AND rosa_enabled = 'true' AND channel_group = 'stable' AND default = 'true'"

	for {
		var response *clustersmgmtv1.VersionsListResponse
		response, err = collection.List().
			Search(filter).
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return versions, fmt.Errorf("unable to list versions at page [%d] - %w", page, err)
		}
		versions = append(versions, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}

	// Sort list in descending order
	sort.Slice(versions, func(i, j int) bool {
		a, erra := ver.NewVersion(versions[i].RawID())
		b, errb := ver.NewVersion(versions[j].RawID())
		if erra != nil || errb != nil {
			return false
		}
		return a.GreaterThan(b)
	})

	return
}

// GetDefaultVersion gets the default (latest) version.
// Copied from https://github.com/openshift/rosa/blob/master/pkg/ocm/versions.go#L219.
func GetDefaultVersion(connection *sdk.Connection) (version *clustersmgmtv1.Version, err error) {
	response, err := GetAvailableVersions(connection)
	if err != nil {
		return version, fmt.Errorf("unable to get available versions - %w", err)
	}

	if len(response) > 0 {
		if response[0] != nil {
			return response[0], nil
		}

	}

	return version, ErrVersionsNotFound
}
