package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Config struct {
	User              string
	FromTime          string
	ToTime            string
	MaxRepos          int
	RepoPattern       string
	OAuth2TokenSource oauth2.TokenSource
}

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

type QueryRoot struct {
	User User `graphql:"user(login: $user)"`
}

type RepoStats struct {
	Commits   int `json:"commits"`
	PRs       int `json:"prs"`
	PRReviews int `json:"prReviews"`
}

type QueryVariables map[string]interface{}

type RepoStatsMap map[string]*RepoStats

func aggregateContributions(statsMap RepoStatsMap, query QueryRoot, repoRegexp *regexp.Regexp) {
	for _, commitContribs := range query.User.ContributionsCollection.CommitContributionsByRepository {
		key := string(commitContribs.Repository.NameWithOwner)
		match := repoRegexp.MatchString(key)

		if match {
			statsMap[key] = &RepoStats{}
			statsMap[key].Commits += int(commitContribs.Contributions.TotalCount)
		}
	}

	for _, prContribs := range query.User.ContributionsCollection.PullRequestContributionsByRepository {
		key := string(prContribs.Repository.NameWithOwner)
		match := repoRegexp.MatchString(key)

		if match {
			if rc2, ok := statsMap[key]; ok {
				rc2.PRs += int(prContribs.Contributions.TotalCount)
			} else {
				statsMap[key] = &RepoStats{}
				statsMap[key].PRs += int(prContribs.Contributions.TotalCount)
			}
		}
	}

	for _, reviewContribs := range query.User.ContributionsCollection.PullRequestReviewContributionsByRepository {
		key := string(reviewContribs.Repository.NameWithOwner)
		match := repoRegexp.MatchString(key)

		if match {
			if rc2, ok := statsMap[key]; ok {
				rc2.PRReviews += int(reviewContribs.Contributions.TotalCount)
			} else {
				statsMap[key] = &RepoStats{}
				statsMap[key].PRReviews += int(reviewContribs.Contributions.TotalCount)
			}
		}
	}
}

func renderTable(statsMap RepoStatsMap, cfg *Config, userName githubv4.String) {
	totalCommits := 0
	totalPRs := 0
	totalReviews := 0
	table := tablewriter.NewWriter(os.Stdout)

	fmt.Printf("User: %s\nName: %s\nStart: %s\nStop: %s\n", cfg.User, userName, cfg.FromTime, cfg.ToTime)
	table.SetHeader([]string{"repo", "commits", "prs", "pr_reviews"})

	for k, v := range statsMap {
		table.Append([]string{k, fmt.Sprint(v.Commits), fmt.Sprint(v.PRs), fmt.Sprint(v.PRReviews)})
		totalCommits += v.Commits
		totalPRs += v.PRs
		totalReviews += v.PRReviews
	}

	table.SetFooterAlignment(2)
	table.SetFooter([]string{"totals", fmt.Sprint(totalCommits), fmt.Sprint(totalPRs), fmt.Sprint(totalReviews)})
	table.Render()
}

func run(cfg *Config) error {
	httpClient := oauth2.NewClient(context.Background(), cfg.OAuth2TokenSource)
	client := githubv4.NewClient(httpClient)
	repoPattern := cfg.RepoPattern
	repoRegexp := regexp.MustCompile(repoPattern)

	fromTime, err := time.Parse(time.RFC3339, cfg.FromTime)
	if err != nil {
		return err
	}

	toTime, err := time.Parse(time.RFC3339, cfg.ToTime)
	if err != nil {
		return err
	}

	variables := map[string]interface{}{
		"user":     githubv4.String(cfg.User),
		"maxRepos": githubv4.Int(cfg.MaxRepos),
		"fromTime": githubv4.DateTime{Time: fromTime},
		"toTime":   githubv4.DateTime{Time: toTime},
	}

	var query QueryRoot

	err = client.Query(context.Background(), &query, variables)
	if err != nil {
		return err
	}

	repoStats := make(map[string]*RepoStats)

	aggregateContributions(repoStats, query, repoRegexp)

	renderTable(repoStats, cfg, query.User.Name)

	return nil
}

func main() {
	cfg := new(Config)
	flag.StringVar(&cfg.User, "user", "mble-sfdc", "user to query stats for")
	flag.StringVar(&cfg.FromTime, "since", time.Now().Add(-time.Hour*24*30).Format(time.RFC3339), "RFC3339 timestamp to start query from")
	flag.StringVar(&cfg.ToTime, "until", time.Now().Format(time.RFC3339), "RFC3339 timestamp to start query until")
	flag.StringVar(&cfg.RepoPattern, "pattern", ".*", "Go regexp to restrict counted repos")
	flag.IntVar(&cfg.MaxRepos, "max-repos", 25, "max repos to query")

	flag.Parse()

	cfg.OAuth2TokenSource = oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)

	err := run(cfg)
	if err != nil {
		log.Fatalf("FATAL: %s", err)
	}
}
