package query

import (
	"github.com/shurcooL/githubv4"
)

type PullRequestContributionsByRepository []struct {
	Repository struct {
		NameWithOwner githubv4.String
	}
	Contributions struct {
		TotalCount githubv4.Int
	}
}

type PullRequestReviewContributionsByRepository []struct {
	Repository struct {
		NameWithOwner githubv4.String
	}
	Contributions struct {
		TotalCount githubv4.Int
	}
}

type CommitContributionsByRepository []struct {
	Repository struct {
		NameWithOwner githubv4.String
	}
	Contributions struct {
		TotalCount githubv4.Int
	}
}

type ContributionsCollection struct {
	CommitContributionsByRepository            CommitContributionsByRepository            `graphql:"commitContributionsByRepository(maxRepositories: $maxRepos)"`
	PullRequestContributionsByRepository       PullRequestContributionsByRepository       `graphql:"pullRequestContributionsByRepository(maxRepositories: $maxRepos)"`
	PullRequestReviewContributionsByRepository PullRequestReviewContributionsByRepository `graphql:"pullRequestReviewContributionsByRepository(maxRepositories: $maxRepos)"`
}

type User struct {
	Login                   githubv4.String
	Name                    githubv4.String
	ContributionsCollection ContributionsCollection `graphql:"contributionsCollection(from: $fromTime, to: $toTime)"`
}

type Root struct {
	User User `graphql:"user(login: $user)"`
}

type TeamMember struct {
	Login githubv4.String
}

type Members struct {
	Nodes []TeamMember `graphql:"nodes"`
}
type Team struct {
	Members Members
}

type Organization struct {
	Login githubv4.String
	Name  githubv4.String
	Team  Team `graphql:"team(slug: $team)"`
}

type Org struct {
	Organization Organization `graphql:"organization(login: $org)"`
}
