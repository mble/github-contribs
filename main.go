package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mble/github-contribs/query"
	"github.com/olekukonko/tablewriter"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Team              string
	Users             []string
	FromTime          string
	ToTime            string
	MaxRepos          int
	RepoPattern       string
	OAuth2TokenSource oauth2.TokenSource
}

type RepoStats struct {
	Commits   int `json:"commits"`
	PRs       int `json:"prs"`
	PRReviews int `json:"prReviews"`
}

type RepoStatsMap map[string]*RepoStats

type UserStats struct {
	Login        string
	Name         string
	RepoStatsMap RepoStatsMap
}

func aggregateContributions(userStats UserStats, q query.Root, repoRegexp *regexp.Regexp) {
	statsMap := userStats.RepoStatsMap

	for _, commitContribs := range q.User.ContributionsCollection.CommitContributionsByRepository {
		key := string(commitContribs.Repository.NameWithOwner)
		match := repoRegexp.MatchString(key)

		if match {
			statsMap[key] = &RepoStats{}
			statsMap[key].Commits += int(commitContribs.Contributions.TotalCount)
		}
	}

	for _, prContribs := range q.User.ContributionsCollection.PullRequestContributionsByRepository {
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

	for _, reviewContribs := range q.User.ContributionsCollection.PullRequestReviewContributionsByRepository {
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

func renderTable(userStats UserStats, cfg *Config) {
	totalCommits := 0
	totalPRs := 0
	totalReviews := 0
	table := tablewriter.NewWriter(os.Stdout)

	fmt.Printf("User: %s\nName: %s\nStart: %s\nStop: %s\n", userStats.Login, userStats.Name, cfg.FromTime, cfg.ToTime)
	table.SetHeader([]string{"repo", "commits", "prs", "pr_reviews"})

	for k, v := range userStats.RepoStatsMap {
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

	var orgQuery query.Org

	orgTeam := strings.Split(cfg.Team, "/")
	variables := map[string]interface{}{
		"org":  githubv4.String(orgTeam[0]),
		"team": githubv4.String(orgTeam[1]),
	}

	err = client.Query(context.Background(), &orgQuery, variables)
	if err != nil {
		return err
	}

	members := []string{}
	for _, member := range orgQuery.Organization.Team.Members.Nodes {
		members = append(members, string(member.Login))
	}

	sort.SliceStable(members, func(i, j int) bool { return members[i] < members[j] })

	stats := make([]UserStats, len(members))
	g, ctx := errgroup.WithContext(context.Background())

	for idx, user := range members {
		idx, user := idx, user
		variables := map[string]interface{}{
			"user":     githubv4.String(user),
			"maxRepos": githubv4.Int(cfg.MaxRepos),
			"fromTime": githubv4.DateTime{Time: fromTime},
			"toTime":   githubv4.DateTime{Time: toTime},
		}

		g.Go(func() error {

			var query query.Root

			err = client.Query(ctx, &query, variables)
			if err != nil {
				return err
			}

			userStats := UserStats{
				Login:        user,
				Name:         string(query.User.Name),
				RepoStatsMap: make(RepoStatsMap),
			}

			aggregateContributions(userStats, query, repoRegexp)

			stats[idx] = userStats
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for _, stat := range stats {
		renderTable(stat, cfg)
	}

	return nil
}

func main() {
	cfg := new(Config)

	var users string

	flag.StringVar(&cfg.Team, "team", "", "GitHub team in org/team-slug format. Mutally exclusive with -users")
	flag.StringVar(&users, "users", "", "comma-separated list of users to query. Mutually exclusive with -team")
	flag.StringVar(&cfg.FromTime, "since", time.Now().Add(-time.Hour*24*30).Format(time.RFC3339), "RFC3339 timestamp to start query from")
	flag.StringVar(&cfg.ToTime, "until", time.Now().Format(time.RFC3339), "RFC3339 timestamp to start query until")
	flag.StringVar(&cfg.RepoPattern, "pattern", ".*", "Go regexp to restrict counted repos")
	flag.IntVar(&cfg.MaxRepos, "max-repos", 25, "max repos to query")

	flag.Parse()

	if users != "" && cfg.Team != "" {
		log.Fatalln("FATAL: cannot pass in both teams and a user list.")
	}

	if users == "" && cfg.Team == "" {
		log.Fatalln("FATAL: must pass in either a user list or a team.")
	}

	cfg.Users = strings.Split(users, ",")
	cfg.OAuth2TokenSource = oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)

	err := run(cfg)
	if err != nil {
		log.Fatalf("FATAL: %s", err)
	}
}
