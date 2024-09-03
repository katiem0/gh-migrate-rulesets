package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/shurcooL/graphql"
	"go.uber.org/zap"
)

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

func GetAuthToken(token, hostname string) string {
	if token != "" {
		return token
	}
	t, _ := auth.TokenForHost(hostname)
	return t
}

type Getter interface {
	GetAppInstallations(owner string) ([]byte, error)
	GetCustomRoles(owner string, roleID int) ([]byte, error)
	GetRepo(owner string, name string) ([]data.RepoSingleQuery, error)
	GetRepoByID(repoID int) (*data.RepoInfo, error)
	GetReposList(owner string, endCursor *string) ([]data.ReposQuery, error)
	GetOrgRulesetsList(owner string, endCursor *string) (*data.OrgRulesetsQuery, error)
	GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error)
	GetRepoRulesetsList(owner string, endCursor *string) (*data.RepoRulesetsQuery, error)
	GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error)
	GetTeamData(ownerID int, teamID int) (*data.TeamInfo, error)
	GetTeamByName(owner string, teamSlug string) (*data.TeamInfo, error)
	CreateOrgLevelRuleset(owner string, data io.Reader)
	CreateRepoLevelRuleset(ownerRepo string, data io.Reader)
	FetchOrgId(owner string) (*data.OrgIdQuery, error)
	FetchOrgRulesets(owner string) ([]data.Rulesets, error)
	GatherRepositories(owner string, repos []string) []data.RepoInfo
	RepoExists(ownerRepo string) bool
	ParseBypassActorsForImport(owner string, bypassActorsStr string) []data.BypassActor
	UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s *APIGetter) data.RepoRuleset
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

func (g *APIGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	url := fmt.Sprintf("orgs/%s/rulesets", owner)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (g *APIGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	url := fmt.Sprintf("repos/%s/rulesets", ownerRepo)

	resp, err := g.restClient.Request("POST", url, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (g *APIGetter) FetchOrgId(owner string) (*data.OrgIdQuery, error) {
	query := new(data.OrgIdQuery)
	variables := map[string]interface{}{
		"owner": graphql.String(owner),
	}
	err := g.gqlClient.Query("getOrgRulesets", &query, variables)
	return query, err
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
		zap.S().Debugf("Checking for rulesets in repo %s", repo.Name)
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
func (g *APIGetter) GetAnApp(appSlug string) (*data.AppInfo, error) {
	url := fmt.Sprintf("apps/%s", appSlug)
	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var appInfo data.AppInfo
	err = json.Unmarshal(responseData, &appInfo)
	if err != nil {
		return nil, err
	}
	return &appInfo, nil
}

func (g *APIGetter) GetAppInstallations(owner string) (*data.AppIntegrations, error) {
	var allInstallations data.AppIntegrations
	url := fmt.Sprintf("orgs/%s/installations?per_page=100", owner)
	for {
		resp, err := g.restClient.Request("GET", url, nil)
		if err != nil {
			zap.S().Error("Error raised in getting app installations", zap.Error(err))
			return nil, err
		}
		defer resp.Body.Close()
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			zap.S().Error("Error reading response body", zap.Error(err))
			return nil, err
		}
		var tempInstallations data.AppIntegrations
		err = json.Unmarshal(responseData, &tempInstallations)
		if err != nil {
			zap.S().Error("Error unmarshalling response data", zap.Error(err))
			return nil, err
		}
		allInstallations.TotalCount = tempInstallations.TotalCount
		allInstallations.Installations = append(allInstallations.Installations, tempInstallations.Installations...)

		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}
		nextURL := getNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}
		url = nextURL
	}
	return &allInstallations, nil
}

func getNextPageURL(linkHeader string) string {
	const prefix = "https://api.github.com/"
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 {
			continue
		}
		urlPart := strings.Trim(parts[0], "<>")
		relPart := strings.TrimSpace(parts[1])
		if relPart == `rel="next"` {
			return strings.TrimPrefix(urlPart, prefix)
		}
	}
	return ""
}

func (g *APIGetter) GetCustomRoles(owner string, roleID int) (*data.CustomRole, error) {
	url := fmt.Sprintf("orgs/%s/custom-repository-roles/%s", owner, strconv.Itoa(roleID))

	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var roleName data.CustomRole
	err = json.Unmarshal(responseData, &roleName)
	return &roleName, err
}
func (g *APIGetter) GetRepoCustomRoles(owner string) (*data.CustomRepoRoles, error) {
	var allCustomRoles data.CustomRepoRoles
	url := fmt.Sprintf("orgs/%s/custom-repository-roles?per_page=100", owner)
	for {
		resp, err := g.restClient.Request("GET", url, nil)
		if err != nil {
			zap.S().Error("Error raised in getting repo custom roles", zap.Error(err))
			return nil, err
		}
		defer resp.Body.Close()
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			zap.S().Error("Error reading response body", zap.Error(err))
			return nil, err
		}
		var tempCustomRoles data.CustomRepoRoles
		err = json.Unmarshal(responseData, &tempCustomRoles)
		if err != nil {
			zap.S().Error("Error unmarshalling response data", zap.Error(err))
			return nil, err
		}
		allCustomRoles.TotalCount = tempCustomRoles.TotalCount
		allCustomRoles.CustomRoles = append(allCustomRoles.CustomRoles, tempCustomRoles.CustomRoles...)

		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}
		nextURL := getNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}
		url = nextURL
	}
	return &allCustomRoles, nil
}

func (g *APIGetter) GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error) {
	url := fmt.Sprintf("orgs/%s/rulesets/%s", owner, strconv.Itoa(rulesetId))

	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return responseData, nil
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

func (g *APIGetter) GetRepo(owner string, name string) (*data.RepoSingleQuery, error) {
	query := new(data.RepoSingleQuery)
	variables := map[string]interface{}{
		"owner": graphql.String(owner),
		"name":  graphql.String(name),
	}

	err := g.gqlClient.Query("getRepo", &query, variables)
	return query, err
}

func (g *APIGetter) GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error) {
	url := fmt.Sprintf("repos/%s/%s/rulesets/%s", owner, repo, strconv.Itoa(rulesetId))

	resp, _ := g.restClient.Request("GET", url, nil)
	defer resp.Body.Close()

	responseData, _ := io.ReadAll(resp.Body)
	return responseData, nil
}

func (g *APIGetter) GetRepoByID(repoID int) (*data.RepoInfo, error) {
	url := fmt.Sprintf("repositories/%s", strconv.Itoa(repoID))

	resp, _ := g.restClient.Request("GET", url, nil)
	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repoInfo data.RepoInfo
	err = json.Unmarshal(responseData, &repoInfo)
	return &repoInfo, err
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

func (g *APIGetter) GetTeamData(ownerID int, teamID int) (*data.TeamInfo, error) {
	url := fmt.Sprintf("organizations/%s/team/%s", strconv.Itoa(ownerID), strconv.Itoa(teamID))

	resp, _ := g.restClient.Request("GET", url, nil)
	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teamName data.TeamInfo
	err = json.Unmarshal(responseData, &teamName)
	return &teamName, err
}

func (g *APIGetter) GetTeamByName(owner string, teamSlug string) (*data.TeamInfo, error) {
	url := fmt.Sprintf("orgs/%s/teams/%s", owner, teamSlug)

	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teamName data.TeamInfo
	err = json.Unmarshal(responseData, &teamName)
	if err != nil {
		return nil, err
	}
	return &teamName, nil
}

func (g *APIGetter) RepoExists(ownerRepo string) bool {
	url := fmt.Sprintf("repos/%s", ownerRepo)
	resp, err := g.restClient.Request("GET", url, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false
		}
		return false
	}
	defer resp.Body.Close()
	return true
}
