package lib

import (
	"context"

	githubql "github.com/shurcooL/githubv4"
	"github.com/shurcooL/graphql"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

type githubExtendClient interface {
	Query(context.Context, interface{}, map[string]interface{}) error
}

type MemberNode struct {
	Login graphql.String
}

type MemberEdge struct {
	Node MemberNode
}

type Members struct {
	Edges    []MemberEdge
	PageInfo struct {
		HasNextPage githubql.Boolean
		EndCursor   githubql.String
	}
}

type Team struct {
	Members Members `graphql:"members(first: 100, after: $cursor)"`
}

type TeamMembersQuery struct {
	RateLimit struct {
		Cost      githubql.Int
		Remaining githubql.Int
	}
	Organization struct {
		Team Team `graphql:"team(slug: $teamSlug)"`
	} `graphql:"organization(login: $org)"`
}

func ListTeamAllMembers(ctx context.Context, log *logrus.Entry, ghc githubExtendClient,
	org, teamSlug string) ([]github.TeamMember, error) {
	var ret []github.TeamMember
	vars := map[string]interface{}{
		"org":      githubql.String(org),
		"teamSlug": githubql.String(teamSlug),
		"cursor":   (*githubql.String)(nil),
	}
	var totalCost int
	var remaining int
	for {
		sq := TeamMembersQuery{}
		if err := ghc.Query(ctx, &sq, vars); err != nil {
			return nil, err
		}
		totalCost += int(sq.RateLimit.Cost)
		remaining = int(sq.RateLimit.Remaining)
		for _, e := range sq.Organization.Team.Members.Edges {
			ret = append(ret, github.TeamMember{Login: string(e.Node.Login)})
		}
		pageInfo := sq.Organization.Team.Members.PageInfo
		if !pageInfo.HasNextPage {
			break
		}
		vars["cursor"] = githubql.NewString(pageInfo.EndCursor)
	}
	log.Infof("List members for org:%s team:%s cost %d point(s). %d remaining.", org, teamSlug, totalCost, remaining)
	return ret, nil
}
