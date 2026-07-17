package list

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/katiem0/gh-migrate-rulesets/internal/log"
	"github.com/katiem0/gh-migrate-rulesets/internal/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type cmdFlags struct {
	token    string
	hostname string
	listFile string
	ruleType string
	debug    bool
}

type ListGetter interface {
	FetchOrgId(owner string) (*data.OrgIdQuery, error)
	FetchOrgRulesets(owner string) ([]data.Rulesets, error)
	FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error)
	GatherRepositories(owner string, repos []string) []data.RepoInfo
	GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error)
	GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error)
	ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string
	ProcessRules(rules []data.Rules) map[string]string
}

// buildRuleColumns returns the rule values in registry (CSV column) order,
// keyed by rule type. Empty string is emitted for rule types not present.
func buildRuleColumns(rulesMap map[string]string) []string {
	columns := make([]string, 0, len(data.RuleRegistry))
	for _, spec := range data.RuleRegistry {
		columns = append(columns, rulesMap[spec.Type])
	}
	return columns
}

func NewCmdList() *cobra.Command {
	cmdFlags := cmdFlags{}
	var authToken string

	listCmd := &cobra.Command{
		Use:   "list [flags] <organization> [repo ...]",
		Short: "Generate a report of rulesets for repositories and/or organization.",
		Long:  "Generate a report of rulesets for a list of repositories and/or organization.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(listCmd *cobra.Command, args []string) error {
			logger, _ := log.NewLogger(cmdFlags.debug)
			defer logger.Sync() // nolint:errcheck
			zap.ReplaceGlobals(logger)

			zap.S().Infof("Starting ruleset listing process")
			zap.S().Debugf("Command arguments: organization=%s, repos=%v", args[0], args[1:])
			zap.S().Debugf("Flags: hostname=%s, ruleType=%s, output-file=%s, debug=%t",
				cmdFlags.hostname, cmdFlags.ruleType, cmdFlags.listFile, cmdFlags.debug)

			validRuleTypes := map[string]struct{}{
				"all":      {},
				"repoOnly": {},
				"orgOnly":  {},
			}
			if _, isValid := validRuleTypes[cmdFlags.ruleType]; !isValid {
				return fmt.Errorf("invalid ruleType: %s. Valid values are 'all', 'repoOnly', or 'orgOnly'", cmdFlags.ruleType)
			}

			zap.S().Debugf("Authenticating with hostname: %s", cmdFlags.hostname)
			authToken = utils.GetAuthToken(cmdFlags.token, cmdFlags.hostname)
			if authToken == "" {
				return fmt.Errorf("failed to retrieve authentication token for hostname: %s", cmdFlags.hostname)
			}
			zap.S().Debug("Authentication token retrieved successfully")

			zap.S().Debug("Initializing GitHub API clients")
			restClient, gqlClient, err := utils.InitializeClients(cmdFlags.hostname, authToken)
			if err != nil {
				zap.S().Errorf("Failed to initialize API clients: %v", err)
				return err
			}
			zap.S().Info("Successfully initialized API clients")

			owner := args[0]
			repos := args[1:]

			if len(repos) > 0 {
				zap.S().Infof("Processing %d specific repositories: %v", len(repos), repos)
			} else {
				zap.S().Info("Processing all repositories in organization")
			}

			return runCmdList(owner, repos, &cmdFlags, utils.NewAPIGetter(gqlClient, restClient), cmdFlags.listFile)
		},
	}

	reportFileDefault := fmt.Sprintf("ruleset-%s.csv", time.Now().Format("20060102150405"))
	ruleDefault := "all"

	listCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub Personal Access Token (default "gh auth token")`)
	listCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GHES (github.example.com), or data residency (tenant.ghe.com)")
	listCmd.Flags().StringVarP(&cmdFlags.listFile, "output-file", "o", reportFileDefault, "Name of file to write CSV list to")
	listCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")
	listCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")
	return listCmd
}

func runCmdList(owner string, repos []string, cmdFlags *cmdFlags, g ListGetter, outputFileName string) error {
	zap.S().Infof(" Starting ruleset listing for organization: %s", owner)
	zap.S().Debugf("Rule type filter: %s", cmdFlags.ruleType)

	var orgID int
	var processedOrgRulesets int
	var processedRepoRulesets int
	var skippedRulesets int

	zap.S().Infof("Step 1/5: Fetching organization ID for %s", owner)
	orgIDData, err := g.FetchOrgId(owner)
	if err != nil {
		zap.S().Errorf("Failed to fetch organization ID for %s: %v", owner, err)
		return fmt.Errorf("error fetching organization ID: %w", err)
	}
	orgID = orgIDData.Organization.DatabaseID
	zap.S().Infof("Organization ID retrieved: %d", orgID)

	zap.S().Info("Step 2/5: Initializing CSV writer and preparing headers")
	var buffer bytes.Buffer
	csvWriter := csv.NewWriter(&buffer)

	headers := []string{
		"RulesetLevel",
		"SourceRepositoryName",
		"TargetRepositoryName",
		"RuleID",
		"RulesetName",
		"Target",
		"Enforcement",
		"BypassActors",
		"ConditionsRefNameInclude",
		"ConditionsRefNameExclude",
		"ConditionsRepoNameInclude",
		"ConditionsRepoNameExclude",
		"ConditionsRepoNameProtected",
		"ConditionRepoPropertyInclude",
		"ConditionRepoPropertyExclude",
	}
	headers = append(headers, data.RuleHeaders()...)
	headers = append(headers, "CreatedAt", "UpdatedAt")

	if err := csvWriter.Write(headers); err != nil {
		zap.S().Errorf("Failed to write CSV headers: %v", err)
		return fmt.Errorf("error writing CSV headers: %w", err)
	}
	zap.S().Debug("CSV headers prepared successfully")

	if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "orgOnly" {
		zap.S().Infof("Step 3/5: Processing organization-level rulesets for %s", owner)

		allOrgRules, err := g.FetchOrgRulesets(owner)
		if err != nil {
			zap.S().Errorf("Failed to fetch organization rulesets for %s: %v", owner, err)
			if cmdFlags.ruleType == "orgOnly" {
				return fmt.Errorf("error fetching organization rulesets (orgOnly mode): %w", err)
			}
			zap.S().Warn("Continuing with repository-level rulesets...")
		} else {
			zap.S().Infof("Found %d organization-level rulesets to process", len(allOrgRules))

			for idx, singleRule := range allOrgRules {
				zap.S().Infof("[%d/%d] Processing org ruleset: %s (ID: %s, Database ID: %d)",
					idx+1, len(allOrgRules), singleRule.Name, singleRule.ID, singleRule.DatabaseID)

				err := utils.SafeExecute(func() error {
					zap.S().Debugf("Fetching detailed ruleset data for org ruleset ID: %d", singleRule.DatabaseID)

					orgLevelRulesetResponse, err := g.GetOrgLevelRuleset(owner, singleRule.DatabaseID)
					if err != nil {
						return fmt.Errorf("failed to get org level ruleset: %w", err)
					}

					zap.S().Debugf("Unmarshaling org ruleset response for: %s", singleRule.Name)
					var orgLevelRuleset data.RepoRuleset
					if err = json.Unmarshal(orgLevelRulesetResponse, &orgLevelRuleset); err != nil {
						return fmt.Errorf("failed to unmarshal org ruleset: %w", err)
					}

					zap.S().Debugf("Processing bypass actors for org ruleset: %s (count: %d)",
						singleRule.Name, len(orgLevelRuleset.BypassActors))
					Actors := g.ProcessActorsForExport(orgLevelRuleset.BypassActors, owner, orgID, singleRule.ID)
					zap.S().Debugf("Processed %d bypass actors", len(Actors))

					zap.S().Debugf("Processing conditions for org ruleset: %s", singleRule.Name)
					orgConditions := utils.ProcessConditions(orgLevelRuleset)

					zap.S().Debugf("Processing rules for org ruleset: %s (count: %d)",
						singleRule.Name, len(orgLevelRuleset.Rules))
					rulesMap := g.ProcessRules(orgLevelRuleset.Rules)
					zap.S().Debugf("Processed %d rule types", len(rulesMap))

					zap.S().Debugf("Writing CSV row for org ruleset: %s", singleRule.Name)
					row := []string{
						orgLevelRuleset.SourceType,
						"N/A",
						"N/A",
						strconv.Itoa(orgLevelRuleset.ID),
						orgLevelRuleset.Name,
						orgLevelRuleset.Target,
						orgLevelRuleset.Enforcement,
						strings.Join(Actors, "|"),
						orgConditions.IncludeRefNames,
						orgConditions.ExcludeRefNames,
						orgConditions.IncludeNames,
						orgConditions.ExcludeNames,
						orgConditions.BoolNames,
						strings.Join(orgConditions.PropertyInclude, "|"),
						strings.Join(orgConditions.PropertyExclude, "|"),
					}
					row = append(row, buildRuleColumns(rulesMap)...)
					row = append(row, orgLevelRuleset.CreatedAt, orgLevelRuleset.UpdatedAt)
					return csvWriter.Write(row)
				}, fmt.Sprintf("processing org ruleset %s", singleRule.Name))

				if err != nil {
					zap.S().Warnf("Skipping org ruleset %s due to error: %v", singleRule.Name, err)
					skippedRulesets++
					continue
				}

				processedOrgRulesets++
				zap.S().Infof("Successfully processed org ruleset: %s", singleRule.Name)
			}

			zap.S().Infof("Completed organization-level rulesets: %d processed, %d skipped",
				processedOrgRulesets, skippedRulesets)
		}
	} else {
		zap.S().Info("Step 3/5: Skipping organization-level rulesets (ruleType filter)")
	}
	repoSkippedRulesets := 0
	if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
		zap.S().Infof("Step 4/5: Processing repository-level rulesets for %s", owner)

		zap.S().Debug("Gathering repository list")
		allRepos := g.GatherRepositories(owner, repos)
		if len(allRepos) > 0 {
			zap.S().Infof("Found %d repositories to process", len(allRepos))

			zap.S().Debug("Fetching repository rulesets")
			allRepoRules, err := g.FetchRepoRulesets(owner, allRepos)
			if err != nil {
				zap.S().Errorf("Failed to fetch repository rulesets: %v", err)
				if cmdFlags.ruleType == "repoOnly" {
					return fmt.Errorf("error fetching repository rulesets (repoOnly mode): %w", err)
				}
				zap.S().Warn("Continuing without repository rulesets...")
			} else {
				zap.S().Infof("Found %d repository-level rulesets to process", len(allRepoRules))

				for idx, singleRepoRule := range allRepoRules {
					zap.S().Infof("[%d/%d] Processing repo ruleset: %s/%s - %s (Database ID: %d)",
						idx+1, len(allRepoRules), owner, singleRepoRule.RepoName,
						singleRepoRule.Rule.Name, singleRepoRule.Rule.DatabaseID)

					zap.S().Debugf("Fetching detailed ruleset data for %s/%s ruleset: %s",
						owner, singleRepoRule.RepoName, singleRepoRule.Rule.Name)

					repoLevelRulesetResponse, err := g.GetRepoLevelRuleset(owner, singleRepoRule.RepoName, singleRepoRule.Rule.DatabaseID)
					if err != nil {
						zap.S().Warnf("Skipping ruleset %s for repo %s due to error: %v",
							singleRepoRule.Rule.Name, singleRepoRule.RepoName, err)
						repoSkippedRulesets++
						continue
					}

					zap.S().Debugf("Unmarshaling repo ruleset response for %s/%s: %s",
						owner, singleRepoRule.RepoName, singleRepoRule.Rule.Name)
					var repoLevelRuleset data.RepoRuleset
					if err = json.Unmarshal(repoLevelRulesetResponse, &repoLevelRuleset); err != nil {
						zap.S().Warnf("Skipping ruleset %s for repo %s due to error: %v",
							singleRepoRule.Rule.Name, singleRepoRule.RepoName, err)
						repoSkippedRulesets++
						continue
					}

					zap.S().Debugf("Processing bypass actors for repo ruleset: %s (count: %d)",
						singleRepoRule.Rule.Name, len(repoLevelRuleset.BypassActors))
					Actors := g.ProcessActorsForExport(repoLevelRuleset.BypassActors, owner, orgID, singleRepoRule.Rule.ID)
					zap.S().Debugf("Processed %d bypass actors", len(Actors))

					zap.S().Debugf("Processing rules for repo ruleset: %s (count: %d)",
						singleRepoRule.Rule.Name, len(repoLevelRuleset.Rules))
					repoRulesMap := g.ProcessRules(repoLevelRuleset.Rules)
					zap.S().Debugf("Processed %d rule types", len(repoRulesMap))

					zap.S().Debugf("Processing conditions for repo ruleset: %s", singleRepoRule.Rule.Name)
					repoConditions := utils.ProcessConditions(repoLevelRuleset)

					zap.S().Debugf("Writing CSV row for repo ruleset: %s/%s - %s",
						owner, singleRepoRule.RepoName, singleRepoRule.Rule.Name)
					repoRow := []string{
						repoLevelRuleset.SourceType,
						singleRepoRule.RepoName,
						singleRepoRule.RepoName,
						strconv.Itoa(repoLevelRuleset.ID),
						repoLevelRuleset.Name,
						repoLevelRuleset.Target,
						repoLevelRuleset.Enforcement,
						strings.Join(Actors, "|"),
						repoConditions.IncludeRefNames,
						repoConditions.ExcludeRefNames,
						repoConditions.IncludeNames,
						repoConditions.ExcludeNames,
						repoConditions.BoolNames,
						strings.Join(repoConditions.PropertyInclude, "|"),
						strings.Join(repoConditions.PropertyExclude, "|"),
					}
					repoRow = append(repoRow, buildRuleColumns(repoRulesMap)...)
					repoRow = append(repoRow, repoLevelRuleset.CreatedAt, repoLevelRuleset.UpdatedAt)
					err = csvWriter.Write(repoRow)
					if err != nil {
						zap.S().Warnf("Skipping ruleset %s for repo %s due to error: %v",
							singleRepoRule.Rule.Name, singleRepoRule.RepoName, err)
						repoSkippedRulesets++
						continue
					}

					processedRepoRulesets++
					zap.S().Infof("Successfully processed repo ruleset: %s/%s - %s",
						owner, singleRepoRule.RepoName, singleRepoRule.Rule.Name)
				}

				zap.S().Infof("Completed repository-level rulesets: %d processed, %d skipped",
					processedRepoRulesets, repoSkippedRulesets)
			}
		} else {
			zap.S().Warnf("No repositories found to gather for %s", owner)
			if cmdFlags.ruleType == "repoOnly" {
				return fmt.Errorf("no repositories found (repoOnly mode)")
			}
			zap.S().Warn("Continuing without repository rulesets...")
		}
	} else {
		zap.S().Info("Step 4/5: Skipping repository-level rulesets (ruleType filter)")
	}

	zap.S().Info("Step 5/5: Finalizing output")
	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		zap.S().Errorf("Error flushing CSV writer: %v", err)
		return fmt.Errorf("error finalizing CSV output: %w", err)
	}

	totalSkipped := skippedRulesets + repoSkippedRulesets
	totalProcessed := processedOrgRulesets + processedRepoRulesets

	zap.S().Infof("=== Summary ===")
	zap.S().Infof("Organization: %s", owner)
	zap.S().Infof("Organization rulesets processed: %d", processedOrgRulesets)
	zap.S().Infof("Repository rulesets processed: %d", processedRepoRulesets)
	zap.S().Infof("Total rulesets processed: %d", totalProcessed)
	if totalSkipped > 0 {
		zap.S().Warnf("Rulesets skipped due to errors: %d", totalSkipped)
	}

	if totalProcessed > 0 {
		zap.S().Infof("Creating output file: %s", outputFileName)
		reportWriter, err := os.OpenFile(outputFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			zap.S().Errorf("Failed to create output file %s: %v", outputFileName, err)
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer func() {
			if err := reportWriter.Close(); err != nil {
				zap.S().Errorf("Error closing output file: %v", err)
			}
		}()

		if _, err := buffer.WriteTo(reportWriter); err != nil {
			zap.S().Errorf("Failed to write data to output file: %v", err)
			return fmt.Errorf("error writing to output file: %w", err)
		}

		zap.S().Infof("Output file: %s", outputFileName)
		zap.S().Infof("Completed successfully with %d rulesets", totalProcessed)
	} else {
		if totalSkipped > 0 {
			zap.S().Warn("No rulesets were successfully processed")
			return fmt.Errorf("no rulesets were processed successfully, output file not created")
		} else {
			zap.S().Warn("No rulesets found to process")
			return fmt.Errorf("no rulesets found, output file not created")
		}
	}

	zap.S().Infof("=== End of process ===")

	return nil
}
