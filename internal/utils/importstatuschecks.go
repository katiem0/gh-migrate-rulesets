package utils

import (
	"errors"
	"fmt"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) UpdateStatusCheckIntegrationID(sourceOrg string, ruleset data.RepoRuleset, s Getter) (data.RepoRuleset, error) {
	var errs error
	// Only look up the source org's apps if a check actually has an integration ID.
	var sourceAppIntegration *data.AppIntegrations
	for i, rule := range ruleset.Rules {
		if rule.Type == "required_status_checks" && rule.Parameters != nil {
			for j, statusCheck := range rule.Parameters.RequiredStatusChecks {
				if statusCheck.IntegrationID == nil || *statusCheck.IntegrationID == 0 {
					continue
				}
				if sourceAppIntegration == nil {
					fetched, err := s.GetAppInstallations(sourceOrg)
					if err != nil {
						zap.S().Errorf("Failed to get integration app data for status checks in %s: %v", sourceOrg, err)
						errs = errors.Join(errs, fmt.Errorf("status check: looking up integrations in source org %s: %w", sourceOrg, err))
						break
					}
					sourceAppIntegration = fetched
				}
				zap.S().Debugf("Gathering target integration ID for status check %s", statusCheck.Context)
				matched := false
				for _, app := range sourceAppIntegration.Installations {
					zap.S().Debugf("Processing status check integration %s", app.AppSlug)
					if app.AppID == *statusCheck.IntegrationID {
						matched = true
						appIntegrationInfo, err := g.GetAnApp(app.AppSlug)
						if err != nil {
							zap.S().Errorf("Failed to get new integration app data for status check integration ID %d: %v", *statusCheck.IntegrationID, err)
							errs = errors.Join(errs, fmt.Errorf("status check %q: looking up integration %q in target org: %w", statusCheck.Context, app.AppSlug, err))
						} else {
							ruleset.Rules[i].Parameters.RequiredStatusChecks[j].IntegrationID = &appIntegrationInfo.AppID
						}
						break
					}
				}
				if !matched {
					errs = errors.Join(errs, fmt.Errorf("status check %q: integration with ID %d not found in source org %s", statusCheck.Context, *statusCheck.IntegrationID, sourceOrg))
				}
			}
		}
	}
	return ruleset, errs
}
