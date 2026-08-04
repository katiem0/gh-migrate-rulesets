package utils

import (
	"errors"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// TranslateRuleset rewrites source IDs to their target-org equivalents. Only the
// --source-org path uses it; --from-file rulesets already reference target IDs.
func TranslateRuleset(g, s Getter, owner, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, actorMapping map[string]int) (data.RepoRuleset, error) {
	// Separate vars so every stage still runs and reports after an earlier failure.
	updated, bypassErr := g.UpdateBypassActorID(owner, sourceOrg, sourceOrgID, ruleset, s, actorMapping)
	updated, workflowErr := g.UpdateRequiredWorkflowRepoID(owner, updated, s)
	updated, statusErr := g.UpdateStatusCheckIntegrationID(sourceOrg, updated, s)
	return updated, errors.Join(bypassErr, workflowErr, statusErr)
}
