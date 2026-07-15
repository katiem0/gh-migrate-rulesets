package utils

import (
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) UpdateStatusCheckIntegrationID(owner string, sourceOrg string, ruleset data.RepoRuleset, s Getter) data.RepoRuleset {
	for i, rule := range ruleset.Rules {
		if rule.Type == "required_status_checks" && rule.Parameters != nil {
			sourceAppIntegration, err := s.GetAppInstallations(sourceOrg)
			if err != nil {
				zap.S().Errorf("Failed to get integration app data for status checks in %s: %v", sourceOrg, err)
				continue
			} else {
				for j, statusCheck := range rule.Parameters.RequiredStatusChecks {
					if statusCheck.IntegrationID == nil || *statusCheck.IntegrationID == 0 {
						continue
					}
					zap.S().Debugf("Gathering target integration ID for status check %s", statusCheck.Context)
					for _, app := range sourceAppIntegration.Installations {
						zap.S().Debugf("Processing status check integration %s", app.AppSlug)
						if app.AppID == *statusCheck.IntegrationID {
							appIntegrationInfo, err := g.GetAnApp(app.AppSlug)
							if err != nil {
								zap.S().Errorf("Failed to get new integration app data for status check integration ID %d: %v", *statusCheck.IntegrationID, err)
								continue
							} else {
								ruleset.Rules[i].Parameters.RequiredStatusChecks[j].IntegrationID = &appIntegrationInfo.AppID
								break
							}
						}
					}
				}
			}
		}
	}
	return ruleset
}
