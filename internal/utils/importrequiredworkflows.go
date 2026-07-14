package utils

import (
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) ParseRequiredWorkflowsForImport(owner string, value interface{}, repoMapping map[string]string) []data.Workflows {
	var workflows []data.Workflows
	v, ok := value.([]map[string]string)
	if !ok {
		zap.S().Error("Invalid type for value")
		return workflows
	}
	for _, workflowMap := range v {
		targetRepoName := ResolveTargetRepo(repoMapping, workflowMap["RepositoryName"])
		zap.S().Debugf("Gathering target repository %s ID for each workflow", targetRepoName)

		workflowRepoQuery, err := g.GetRepo(owner, targetRepoName)
		if err != nil {
			zap.S().Error("Failed to get repository data for workflow")
			continue
		} else {
			workflow := data.Workflows{
				Path:         workflowMap["Path"],
				Ref:          workflowMap["Ref"],
				RepositoryID: workflowRepoQuery.Repository.DatabaseId,
				SHA:          workflowMap["SHA"],
			}
			workflows = append(workflows, workflow)
		}
	}
	return workflows
}

func (g *APIGetter) UpdateRequiredWorkflowRepoID(owner string, ruleset data.RepoRuleset, s Getter, repoMapping map[string]string) data.RepoRuleset {
	for i, rule := range ruleset.Rules {
		if rule.Type == "workflows" {
			for j, workflow := range rule.Parameters.Workflows {
				zap.S().Debugf("Gathering target repository %d ID for each workflow", workflow.RepositoryID)
				sourceWorkflowRepoQuery, err := s.GetRepoByID(workflow.RepositoryID)
				if err != nil {
					zap.S().Error("Failed to get repository data for workflow")
					continue
				} else {
					targetWorkflowRepoName := ResolveTargetRepo(repoMapping, sourceWorkflowRepoQuery.Name)
					workflowRepo, err := g.GetRepo(owner, targetWorkflowRepoName)
					if err != nil {
						zap.S().Error("Failed to get repository data for workflow")
						continue
					}
					ruleset.Rules[i].Parameters.Workflows[j].RepositoryID = workflowRepo.Repository.DatabaseId
				}
			}
		}
	}
	return ruleset
}
