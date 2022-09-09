package main

import (
	"context"
	"time"

	gc "github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/github"

	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/cherrypicker"
)

const (
	timeoutForAddCollaborator = 5 * time.Second
	timeoutForListInvitations = 5 * time.Second
)

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

// AddCollaborator to repository.
func (c *extendGithubClient) AddCollaborator(org, repo, user string, permission github.RepoPermissionLevel) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeoutForAddCollaborator)
	defer cancel()

	options := &gc.RepositoryAddCollaboratorOptions{Permission: string(permission)}
	_, _, err := c.rs.AddCollaborator(ctx, org, repo, user, options)
	return err
}

// ListRepoInvitations list repository invitations.
func (c *extendGithubClient) ListRepoInvitations(org, repo string) ([]*gc.RepositoryInvitation, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeoutForListInvitations)
	defer cancel()

	var invitations []*gc.RepositoryInvitation
	for page, nextPage := 1, 0; nextPage > 0; page++ {
		data, res, err := c.rs.ListInvitations(ctx, org, repo, &gc.ListOptions{PerPage: 100, Page: page})
		if err != nil {
			return nil, err
		}

		invitations = append(invitations, data...)
		if res != nil {
			nextPage = res.NextPage
		}
	}

	return invitations, nil
}
