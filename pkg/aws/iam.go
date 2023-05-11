package aws

//nolint:gosec
import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"time"

	rosa "github.com/openshift/rosa/pkg/aws"
)

const (
	operatorRolesPrefixChars  = "abcdefghijklmnopqrstuvwxyz0123456789"
	operatorRolesPrefixLength = 6
	operatorRolesMaxLength    = 64

	thumprintTimeout  = 10 * time.Second
	thumprintInterval = 1 * time.Second
)

var (
	ErrTimeoutThumbprint = errors.New("timed out waiting for thumbprint")
)

// CreateOIDCProvider creates an IAM OIDC Identity Provider in AWS.  It uses the
// libraries from the rosa CLI to accomplish this in order to maintain consistent
// and supportable behavior.
func (awsClient *Client) CreateOIDCProvider(issuerURL string) (providerARN string, err error) {
	thumbprint, err := waitForThumbprint(issuerURL)
	if err != nil {
		return providerARN, fmt.Errorf("unable to retrieve oidc provider thumbprint - %w", err)
	}

	// create the oidc provider
	providerARN, err = awsClient.CreateOpenIDConnectProvider(issuerURL, thumbprint, "")
	if err != nil {
		return providerARN, fmt.Errorf("unable to create oidc provider - %w", err)
	}

	return providerARN, nil
}

// DeleteOIDCProvider deletes an IAM OIDC Identity Provider from AWS.  It uses the
// libraries from the rosa CLI to accomplish this in order to maintain consistent
// and supportable behavior.
func (awsClient *Client) DeleteOIDCProvider(oidcProviderARN string) error {
	// delete the oidc provider
	if err := awsClient.DeleteOpenIDConnectProvider(oidcProviderARN); err != nil {
		return fmt.Errorf("delete oidc provider - %w", err)
	}

	return nil
}

func GetOperatorRolesPrefixForCluster(cluster string) string {
	var id string

	for i := 0; i < operatorRolesPrefixLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(operatorRolesPrefixChars))))
		if err != nil {
			panic(err)
		}

		id += string(operatorRolesPrefixChars[num.Int64()])
	}

	return fmt.Sprintf("%s-%s", cluster, id)
}

func GetOperatorRoleArn(name, namespace, accountID, prefix string) string {
	role := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(role) > operatorRolesMaxLength {
		role = role[0:operatorRolesMaxLength]
	}

	return fmt.Sprintf("arn:%s:iam::%s:role/%s", rosa.GetPartition(), accountID, role)
}

// waitForThumbprint attempts to get the thumbprint up to a specific timeout value.  This
// is because the thumprint may not be ready by the time we call this function.  We do not
// want to make this a long running function but we do want to give the reconciler sufficient
// time to retrieve the thumbprint without spitting back an error and requeueing.
func waitForThumbprint(oidcEndpointURL string) (thumbprint string, err error) {
	// create the ticker at an interval, ensuring that we stop it to avoid a
	// memory leak.
	ticker := time.NewTicker(thumprintInterval)
	defer ticker.Stop()

	timeout := time.After(thumprintTimeout)

	for {
		select {
		case <-timeout:
			return thumbprint, ErrTimeoutThumbprint
		case <-ticker.C:
			thumbprint, err := getThumbprint(oidcEndpointURL)
			if err != nil {
				continue
			}

			return thumbprint, nil
		}
	}
}

// getThumbprint gets the thumbprint needed to create the OIDC provider in AWS.
// copied from https://github.com/openshift/rosa/blob/master/cmd/create/oidcprovider/cmd.go
func getThumbprint(oidcEndpointURL string) (string, error) {
	connect, err := url.ParseRequestURI(oidcEndpointURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse uri from oidc endpoint url - %w", err)
	}

	response, err := http.Get(fmt.Sprintf("https://%s:443", connect.Host))
	if err != nil {
		return "", fmt.Errorf("unable to get request from host [%s] - %w", connect.Host, err)
	}
	defer response.Body.Close()

	certChain := response.TLS.PeerCertificates

	// grab the CA in the chain
	for _, cert := range certChain {
		if cert.IsCA {
			if bytes.Equal(cert.RawIssuer, cert.RawSubject) {
				return sha1Hash(cert.Raw), nil
			}
		}
	}

	// fall back to using the last certficiate in the chain
	cert := certChain[len(certChain)-1]

	return sha1Hash(cert.Raw), nil
}

// sha1Hash computes the SHA1 of the byte array and returns the hex encoding as a string.
// copied from https://github.com/openshift/rosa/blob/master/cmd/create/oidcprovider/cmd.go
//
//nolint:gosec
func sha1Hash(data []byte) string {
	hasher := sha1.New()
	hasher.Write(data)
	hashed := hasher.Sum(nil)

	return hex.EncodeToString(hashed)
}
