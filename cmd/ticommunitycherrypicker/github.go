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

func newExtGithubClient(client github.Client) cherrypicker.GithubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "... your access token ..."},
	)
	tc := oauth2.NewClient(ctx, ts)
	cc := gc.NewClient(tc)

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
