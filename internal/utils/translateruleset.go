package utils

import "github.com/katiem0/gh-migrate-rulesets/internal/data"

// TranslateRuleset rewrites source IDs (bypass actors, required-workflow repos,
// status-check integrations) to their equivalents in the target org. It is used by
// the live --source-org path; --from-file rulesets are assumed to already reference
// target IDs (only bypass actor mapping is applied during CSV parsing).
func TranslateRuleset(g, s Getter, owner, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, actorMapping map[string]int) data.RepoRuleset {
	updated := g.UpdateBypassActorID(owner, sourceOrg, sourceOrgID, ruleset, s, actorMapping)
	updated = g.UpdateRequiredWorkflowRepoID(owner, updated, s)
	updated = g.UpdateStatusCheckIntegrationID(sourceOrg, updated, s)
	return updated
}
