package identityprovider

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/xanzy/go-gitlab"
)

var (
	ErrMissingApplication = errors.New("unable to find gitlab application")
)

type GitLab struct {
	Client *gitlab.Client
}

func (glc *GitLab) GetApplication(name string) (*gitlab.Application, error) {
	// list all applications
	applications, _, err := glc.Client.Applications.ListApplications(&gitlab.ListApplicationsOptions{})
	if err != nil {
		return &gitlab.Application{}, fmt.Errorf("unable to list gitlab applications - %w", err)
	}

	// find the correct application based on the name
	for i := range applications {
		if applications[i].ApplicationName == name {
			return applications[i], nil
		}
	}

	// return a nil object if we were unable to find any
	return nil, fmt.Errorf("unable to find gitlab application [%s] - %w", name, ErrMissingApplication)
}

func (glc *GitLab) CreateApplication(name, callbackURL string) (*gitlab.Application, error) {
	// create the application
	application, _, err := glc.Client.Applications.CreateApplication(&gitlab.CreateApplicationOptions{
		Name:         gitlab.String(name),
		RedirectURI:  gitlab.String(callbackURL),
		Scopes:       gitlab.String("openid"),
		Confidential: gitlab.Bool(true),
	})
	if err != nil {
		return &gitlab.Application{}, fmt.Errorf("unable to create gitlab application - %w", err)
	}

	return application, nil
}

func (glc *GitLab) UpdateApplication(name, callbackURL string) (*gitlab.Application, error) {
	// create the application
	application, _, err := glc.Client.Applications.CreateApplication(&gitlab.CreateApplicationOptions{
		Name:         gitlab.String(name),
		RedirectURI:  gitlab.String(callbackURL),
		Scopes:       gitlab.String("openid"),
		Confidential: gitlab.Bool(true),
	})
	if err != nil {
		return &gitlab.Application{}, fmt.Errorf("unable to create gitlab application - %w", err)
	}

	return application, nil
}

func EqualGitLab(compare, with gitlab.Application) bool {
	// ignore the id as this is simply the id of the backend object
	compare.ID = with.ID

	return reflect.DeepEqual(compare, with)
}

func DesiredGitLab(name, clientID, clientSecret, callbackURL string, confidential bool) *gitlab.Application {
	return &gitlab.Application{
		ApplicationID:   clientID,
		ApplicationName: name,
		CallbackURL:     callbackURL,
		Secret:          clientSecret,
		Confidential:    confidential,
	}
}
