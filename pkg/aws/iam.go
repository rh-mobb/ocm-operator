package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	rosa "github.com/openshift/rosa/pkg/aws"
	"github.com/sirupsen/logrus"
)

const randomPrefixChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func GetOperatorRolesPrefixForCluster(cluster string) string {
	// create randomizer
	rand.Seed(time.Now().UnixNano())

	// create a random id
	id := make([]byte, 6)
	for i := range id {
		id[i] = randomPrefixChars[rand.Intn(len(randomPrefixChars))]
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
	thumbprint, err := getThumbprint(url)
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
