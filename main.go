package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type CommitContributionsByRepository []struct {
	Repository struct {
		NameWithOwner githubv4.String
	}
	Contributions struct {
		TotalCount githubv4.Int
	}
}

type ContributionsCollection struct {
	CommitContributionsByRepository CommitContributionsByRepository `graphql:"commitContributionsByRepository(maxRepositories: $maxRepos)"`
}

type User struct {
	Login                   githubv4.String
	Name                    githubv4.String
	ContributionsCollection ContributionsCollection `graphql:"contributionsCollection(from: $fromTime, to: $toTime)"`
}

func run() error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)

	user := "mble"
	fromTime, err := time.Parse(time.RFC3339, "2022-01-01T00:00:00Z")
	if err != nil {
		return err
	}
	toTime, err := time.Parse(time.RFC3339, "2022-12-31T23:59:59Z")
	if err != nil {
		return err
	}
	variables := map[string]interface{}{
		"user":     githubv4.String(user),
		"maxRepos": githubv4.Int(20),
		"fromTime": githubv4.DateTime{Time: fromTime},
		"toTime":   githubv4.DateTime{Time: toTime},
	}
	var query struct {
		User User `graphql:"user(login: $user)"`
	}

	err = client.Query(context.Background(), &query, variables)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(query)
	if err != nil {
		return err
	}

	fmt.Println(string(raw))
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}
