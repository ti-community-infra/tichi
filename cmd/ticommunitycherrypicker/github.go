package main

import (
	"context"
	"time"

	gc "github.com/google/go-github/v29/github"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/cherrypicker"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/github"
)

const timeoutForAddCollaborator = 5 * time.Second

type oauth2TokenSource func() []byte

// Token implement interface oauth2.TokenSource.
func (o oauth2TokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(o())}, nil
}

func newExtGithubClient(client github.Client, tokenGenerator oauth2TokenSource) cherrypicker.GithubClient {
	ctx := context.Background()

	ts := oauth2.ReuseTokenSource(nil, tokenGenerator)
	cc := gc.NewClient(oauth2.NewClient(ctx, ts))

	return &extendGithubClient{
		Client: client,
		rs:     cc.Repositories,
	}
}

type extendGithubClient struct {
	github.Client
	rs *gc.RepositoriesService
}

func (c *extendGithubClient) AddCollaborator(org, repo, user, permission string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeoutForAddCollaborator)
	defer cancel()

	options := &gc.RepositoryAddCollaboratorOptions{Permission: permission}
	_, _, err := c.rs.AddCollaborator(ctx, org, repo, user, options)
	return err
}
