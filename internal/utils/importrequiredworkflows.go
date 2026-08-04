package utils

import (
	"errors"
	"fmt"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) ParseRequiredWorkflowsForImport(owner string, value interface{}) []data.Workflows {
	var workflows []data.Workflows
	v, ok := value.([]map[string]string)
	if !ok {
		zap.S().Error("Invalid type for value")
		return workflows
	}
	for _, workflowMap := range v {
		zap.S().Debugf("Gathering target repository %s ID for each workflow", workflowMap["RepositoryName"])

		workflowRepoQuery, err := g.GetRepo(owner, workflowMap["RepositoryName"])
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

func (g *APIGetter) UpdateRequiredWorkflowRepoID(owner string, ruleset data.RepoRuleset, s Getter) (data.RepoRuleset, error) {
	var errs error
	for i, rule := range ruleset.Rules {
		if rule.Type == "workflows" && rule.Parameters != nil {
			for j, workflow := range rule.Parameters.Workflows {
				zap.S().Debugf("Gathering target repository %d ID for each workflow", workflow.RepositoryID)
				sourceWorkflowRepoQuery, err := s.GetRepoByID(workflow.RepositoryID)
				if err != nil {
					zap.S().Error("Failed to get repository data for workflow")
					errs = errors.Join(errs, fmt.Errorf("required workflow: looking up source repository ID %d: %w", workflow.RepositoryID, err))
					continue
				}
				workflowRepo, err := g.GetRepo(owner, sourceWorkflowRepoQuery.Name)
				if err != nil {
					zap.S().Error("Failed to get repository data for workflow")
					errs = errors.Join(errs, fmt.Errorf("required workflow: looking up repository %q in target org %s: %w", sourceWorkflowRepoQuery.Name, owner, err))
					continue
				}
				ruleset.Rules[i].Parameters.Workflows[j].RepositoryID = workflowRepo.Repository.DatabaseId
			}
		}
	}
	return ruleset, errs
}
