package ocm

import (
	"fmt"

	ver "github.com/hashicorp/go-version"
)

func GetVersion(rawVersion string) (string, error) {
	parsedVersion, err := ver.NewVersion(rawVersion)
	if err != nil {
		return "", fmt.Errorf("unable to parse version [%s] - %w", rawVersion, err)
	}

	versionSplit := parsedVersion.Segments64()

	return fmt.Sprintf("%d.%d", versionSplit[0], versionSplit[1]), nil
}
