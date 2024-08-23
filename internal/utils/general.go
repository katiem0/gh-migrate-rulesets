package utils

import (
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/shurcooL/graphql"
	"go.uber.org/zap"
)

func GetAuthToken(token, hostname string) string {
	if token != "" {
		return token
	}
	t, _ := auth.TokenForHost(hostname)
	return t
}

func InitializeClients(hostname, authToken string) (api.RESTClient, api.GQLClient, error) {
	restClient, err := gh.RESTClient(&api.ClientOptions{
		Headers: map[string]string{
			"Accept": "application/vnd.github+json",
		},
		Host:      hostname,
		AuthToken: authToken,
	})
	if err != nil {
		zap.S().Errorf("Error arose retrieving rest client")
		return nil, nil, err
	}

	gqlClient, err := gh.GQLClient(&api.ClientOptions{
		Headers: map[string]string{
			"Accept": "application/vnd.github.hawkgirl-preview+json",
		},
		Host:      hostname,
		AuthToken: authToken,
	})
	if err != nil {
		zap.S().Errorf("Error arose retrieving graphql client")
		return nil, nil, err
	}

	return restClient, gqlClient, nil
}

type Getter interface {
	GetRepo(owner string, name string) ([]data.RepoSingleQuery, error)
	GetReposList(owner string, endCursor *string) ([]data.ReposQuery, error)
	GetOrgRulesetsList(owner string, endCursor *string) (*data.OrgRulesetsQuery, error)
	GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error)
	GetRepoRulesetsList(owner string, endCursor *string) (*data.RepoRulesetsQuery, error)
	GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error)
	CreateOrgLevelRuleset(owner string, data io.Reader)
	CreateRepoLevelRuleset(ownerRepo string, data io.Reader)
	FetchOrgRulesets(owner string) []data.Rulesets
	GatherRepositories(owner string, repos []string) []data.RepoInfo
	RepoExists(ownerRepo string) bool
}

type APIGetter struct {
	gqlClient  api.GQLClient
	restClient api.RESTClient
}

func NewAPIGetter(gqlClient api.GQLClient, restClient api.RESTClient) *APIGetter {
	return &APIGetter{
		gqlClient:  gqlClient,
		restClient: restClient,
	}
}

func (g *APIGetter) GetReposList(owner string, endCursor *string) (*data.ReposQuery, error) {
	query := new(data.ReposQuery)
	variables := map[string]interface{}{
		"endCursor": (*graphql.String)(endCursor),
		"owner":     graphql.String(owner),
	}

	err := g.gqlClient.Query("getRepos", &query, variables)

	return query, err
}

func (g *APIGetter) GetRepo(owner string, name string) (*data.RepoSingleQuery, error) {
	query := new(data.RepoSingleQuery)
	variables := map[string]interface{}{
		"owner": graphql.String(owner),
		"name":  graphql.String(name),
	}

	err := g.gqlClient.Query("getRepo", &query, variables)
	return query, err
}

func (g *APIGetter) GetOrgRulesetsList(owner string, endCursor *string) (*data.OrgRulesetsQuery, error) {
	query := new(data.OrgRulesetsQuery)
	variables := map[string]interface{}{
		"endCursor": (*graphql.String)(endCursor),
		"owner":     graphql.String(owner),
	}

	err := g.gqlClient.Query("getOrgRulesets", &query, variables)

	return query, err
}

func (g *APIGetter) GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error) {
	url := fmt.Sprintf("orgs/%s/rulesets/%s", owner, strconv.Itoa(rulesetId))

	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		log.Printf("Error getting repo level ruleset for %s: %v", owner, err)
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body for repo level ruleset %s: %v", owner, err)
		return nil, err
	}
	return responseData, nil
}

func (g *APIGetter) GetRepoRulesetsList(owner string, repo string, endCursor *string) (*data.RepoRulesetsQuery, error) {
	query := new(data.RepoRulesetsQuery)
	variables := map[string]interface{}{
		"endCursor": (*graphql.String)(endCursor),
		"owner":     graphql.String(owner),
		"name":      graphql.String(repo),
	}

	err := g.gqlClient.Query("getRepoRulesets", &query, variables)

	return query, err
}

func (g *APIGetter) GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error) {
	url := fmt.Sprintf("repos/%s/%s/rulesets/%s", owner, repo, strconv.Itoa(rulesetId))

	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		log.Printf("Error getting repo level ruleset for %s/%s: %v", owner, repo, err)
		return nil, nil
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body for repo level ruleset %s/%s: %v", owner, repo, err)
		return nil, nil
	}
	return responseData, nil
}

func (g *APIGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	url := fmt.Sprintf("orgs/%s/rulesets", owner)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		log.Printf("Error creating org level ruleset for %s: %v", owner, err)
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func (g *APIGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	url := fmt.Sprintf("repos/%s/rulesets", ownerRepo)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		log.Printf("Error creating repo level ruleset for %s: %v\n", ownerRepo, err)
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func (g *APIGetter) FetchOrgRulesets(owner string) ([]data.Rulesets, error) {
	var allOrgRules []data.Rulesets
	var orgRulesCursor *string

	for {
		orgRulesetsQuery, err := g.GetOrgRulesetsList(owner, orgRulesCursor)
		if err != nil {
			zap.S().Error("Error getting organization ruleset list", zap.Error(err))
			return nil, err
		}

		allOrgRules = append(allOrgRules, orgRulesetsQuery.Organization.Rulesets.Nodes...)
		orgRulesCursor = &orgRulesetsQuery.Organization.Rulesets.PageInfo.EndCursor

		if !orgRulesetsQuery.Organization.Rulesets.PageInfo.HasNextPage {
			break
		}
	}

	return allOrgRules, nil
}

func (g *APIGetter) FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error) {
	var allRepoRules []data.RepoNameRule

	for _, repo := range repos {
		var repoRulesCursor *string
		zap.S().Infof("Checking for rulesets in repo %s", repo.Name)
		for {
			repoRulesetsQuery, err := g.GetRepoRulesetsList(owner, repo.Name, repoRulesCursor)
			if err != nil {
				return nil, err
			}
			for _, rule := range repoRulesetsQuery.Repository.Rulesets.Nodes {
				allRepoRules = append(allRepoRules, data.RepoNameRule{RepoName: repo.Name, Rule: rule})
			}
			repoRulesCursor = &repoRulesetsQuery.Repository.Rulesets.PageInfo.EndCursor
			if !repoRulesetsQuery.Repository.Rulesets.PageInfo.HasNextPage {
				break
			}
		}
	}
	return allRepoRules, nil
}

func (g *APIGetter) GatherRepositories(owner string, repos []string) ([]data.RepoInfo, error) {
	var allRepos []data.RepoInfo
	var reposCursor *string

	if len(repos) > 0 {
		for _, repo := range repos {
			repoQuery, err := g.GetRepo(owner, repo)
			if err != nil {
				zap.S().Error("Error raised in getting repo", repo, zap.Error(err))
				continue
			}
			allRepos = append(allRepos, repoQuery.Repository)
		}
	} else {
		for {
			reposQuery, err := g.GetReposList(owner, reposCursor)
			if err != nil {
				zap.S().Error("Error raised in processing list of repos", zap.Error(err))
				return nil, err
			}
			allRepos = append(allRepos, reposQuery.Organization.Repositories.Nodes...)
			reposCursor = &reposQuery.Organization.Repositories.PageInfo.EndCursor
			if !reposQuery.Organization.Repositories.PageInfo.HasNextPage {
				break
			}
		}
	}

	return allRepos, nil
}

func (g *APIGetter) RepoExists(ownerRepo string) bool {
	url := fmt.Sprintf("repos/%s", ownerRepo)
	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false
		}
		log.Printf("Error checking if repo exists for %s: %v\n", ownerRepo, err)
		return false
	}
	defer resp.Body.Close()
	return true
}
