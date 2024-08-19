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
	CreateOrgLevelRuleset(owner string, data io.Reader) error
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
		log.Fatal(err)
	}

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return responseData, err
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
		log.Fatal(err)
	}

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return responseData, err
}

func (g *APIGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	url := fmt.Sprintf("orgs/%s/rulesets", owner)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return err
}

func (g *APIGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	url := fmt.Sprintf("repos/%s/rulesets", ownerRepo)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return err
}
