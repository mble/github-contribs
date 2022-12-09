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
