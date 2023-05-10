package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	rosa "github.com/openshift/rosa/pkg/aws"
	"github.com/sirupsen/logrus"
)

const (
	operatorRolesPrefixChars = "abcdefghijklmnopqrstuvwxyz0123456789"

	thumprintTimeout  = 10 * time.Second
	thumprintInterval = 1 * time.Second
)

var (
	ErrTimeoutThumbprint = errors.New("timed out waiting for thumbprint")
)

func GetOperatorRolesPrefixForCluster(cluster string) string {
	// create randomizer
	rand.Seed(time.Now().UnixNano())

	// create a random id
	id := make([]byte, 6)
	for i := range id {
		id[i] = operatorRolesPrefixChars[rand.Intn(len(operatorRolesPrefixChars))]
	}

	return fmt.Sprintf("%s-%s", cluster, id)
}

func GetOperatorRoleArn(name, namespace, accountID, prefix string) string {
	role := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(role) > 64 {
		role = role[0:64]
	}

	return fmt.Sprintf("arn:%s:iam::%s:role/%s", rosa.GetPartition(), accountID, role)
}

func CreateOIDCProvider(url string) (providerARN string, err error) {
	thumbprint, err := waitForThumbprint(url)
	if err != nil {
		return providerARN, fmt.Errorf("unable to retrieve oidc provider thumbprint - %w", err)
	}

	// create the client
	awsClient, err := rosa.NewClient().Logger(&logrus.Logger{Out: ioutil.Discard}).Build()
	if err != nil {
		return providerARN, fmt.Errorf("unable to create aws client - %w", err)
	}

	// create the oidc provider
	providerARN, err = awsClient.CreateOpenIDConnectProvider(url, thumbprint, "")
	if err != nil {
		return providerARN, fmt.Errorf("unable to create oidc provider - %w", err)
	}

	return providerARN, nil
}

func DeleteOIDCProvider(oidcProviderARN string) error {
	// create the client
	awsClient, err := rosa.NewClient().Logger(&logrus.Logger{Out: ioutil.Discard}).Build()
	if err != nil {
		return fmt.Errorf("unable to create aws client - %w", err)
	}

	// delete the oidc provider
	if err := awsClient.DeleteOpenIDConnectProvider(oidcProviderARN); err != nil {
		return fmt.Errorf("delete oidc provider - %w", err)
	}

	return nil
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
		return "", err
	}

	response, err := http.Get(fmt.Sprintf("https://%s:443", connect.Host))
	if err != nil {
		return "", err
	}

	certChain := response.TLS.PeerCertificates

	// Grab the CA in the chain
	for _, cert := range certChain {
		if cert.IsCA {
			if bytes.Equal(cert.RawIssuer, cert.RawSubject) {
				return sha1Hash(cert.Raw), nil
			}
		}
	}

	// Fall back to using the last certficiate in the chain
	cert := certChain[len(certChain)-1]

	return sha1Hash(cert.Raw), nil
}

// sha1Hash computes the SHA1 of the byte array and returns the hex encoding as a string.
// copied from https://github.com/openshift/rosa/blob/master/cmd/create/oidcprovider/cmd.go
func sha1Hash(data []byte) string {
	// nolint:gosec
	hasher := sha1.New()
	hasher.Write(data)
	hashed := hasher.Sum(nil)

	return hex.EncodeToString(hashed)
}
