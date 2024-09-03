package list

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

			validRuleTypes := map[string]struct{}{
				"all":      {},
				"repoOnly": {},
				"orgOnly":  {},
			}
			if _, isValid := validRuleTypes[cmdFlags.ruleType]; !isValid {
				return fmt.Errorf("invalid ruleType: %s. Valid values are 'all', 'repoOnly', or 'orgOnly'", cmdFlags.ruleType)
			}

			authToken = utils.GetAuthToken(cmdFlags.token, cmdFlags.hostname)
			restClient, gqlClient, err := utils.InitializeClients(cmdFlags.hostname, authToken)
			if err != nil {
				return err
			}

			owner := args[0]
			repos := args[1:]

			if _, err := os.Stat(cmdFlags.listFile); errors.Is(err, os.ErrExist) {
				return err
			}

			reportWriter, err := os.OpenFile(cmdFlags.listFile, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}

			return runCmdList(owner, repos, &cmdFlags, utils.NewAPIGetter(gqlClient, restClient), reportWriter)
		},
	}

	reportFileDefault := fmt.Sprintf("ruleset-%s.csv", time.Now().Format("20060102150405"))
	ruleDefault := "all"

	listCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub Personal Access Token (default "gh auth token")`)
	listCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	listCmd.Flags().StringVarP(&cmdFlags.listFile, "output-file", "o", reportFileDefault, "Name of file to write CSV list to")
	listCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")
	listCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")
	return listCmd
}

func runCmdList(owner string, repos []string, cmdFlags *cmdFlags, g *utils.APIGetter, reportWriter io.Writer) error {
	zap.S().Infof("Gathering repositories and/or rulesets for %s", owner)
	var orgID int

	orgIDData, err := g.FetchOrgId(owner)
	if err != nil {
		zap.S().Error("Error raised in fetching org")
		return err
	} else {
		orgID = orgIDData.Organization.DatabaseID

		csvWriter := csv.NewWriter(reportWriter)
		err = csvWriter.Write([]string{
			"RulesetLevel",
			"RepositoryName",
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
			"RulesCreation",
			"RulesUpdate",
			"RulesDeletion",
			"RulesRequiredLinearHistory",
			"RulesMergeQueue",
			"RulesRequiredDeployments",
			"RulesRequiredSignatures",
			"RulesPullRequest",
			"RulesRequiredStatusChecks",
			"RulesNonFastForward",
			"RulesCommitMessagePattern",
			"RulesCommitAuthorEmailPattern",
			"RulesCommitterEmailPattern",
			"RulesBranchNamePattern",
			"RulesTagNamePattern",
			"RulesFilePathRestriction",
			"RulesFilePathLength",
			"RulesFileExtensionRestriction",
			"RulesMaxFileSize",
			"RulesWorkflows",
			"RulesCodeScanning",
			"CreatedAt",
			"UpdatedAt",
		})

		if err != nil {
			return err
		}

		if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "orgOnly" {
			zap.S().Infof("Gathering organization %s level rulesets", owner)
			allOrgRules, err := g.FetchOrgRulesets(owner)
			if err != nil {
				zap.S().Error("Error raised in fetching org  ruleset data for %s", owner)
			}

			for _, singleRule := range allOrgRules {
				zap.S().Debugf("Gathering specific ruleset data for org rule %s", singleRule.Name)
				orgLevelRulesetResponse, err := g.GetOrgLevelRuleset(owner, singleRule.DatabaseID)
				if err != nil {
					zap.S().Error("Error raised in getting org level ruleset data for %s", singleRule.DatabaseID)
					continue
				} else {
					var orgLevelRuleset data.RepoRuleset
					err = json.Unmarshal(orgLevelRulesetResponse, &orgLevelRuleset)
					if err != nil {
						zap.S().Error("Error raised with variable response")
						continue
					}
					Actors := g.ProcessActorsForExport(orgLevelRuleset.BypassActors, owner, orgID, singleRule.ID)
					orgConditions := utils.ProcessConditions(orgLevelRuleset)
					rulesMap := g.ProcessRules(orgLevelRuleset.Rules)

					zap.S().Debugf("Writing output for org rule %s", singleRule.Name)

					err = csvWriter.Write([]string{
						orgLevelRuleset.SourceType,
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
						rulesMap["creation"],
						rulesMap["update"],
						rulesMap["deletion"],
						rulesMap["required_linear_history"],
						rulesMap["merge_queue"],
						rulesMap["required_deployments"],
						rulesMap["required_signatures"],
						rulesMap["pull_request"],
						rulesMap["required_status_checks"],
						rulesMap["non_fast_forward"],
						rulesMap["commit_message_pattern"],
						rulesMap["commit_author_email_pattern"],
						rulesMap["committer_email_pattern"],
						rulesMap["branch_name_pattern"],
						rulesMap["tag_name_pattern"],
						rulesMap["file_path_restriction"],
						rulesMap["max_file_path_length"],
						rulesMap["file_extension_restriction"],
						rulesMap["max_file_size"],
						rulesMap["workflows"],
						rulesMap["code_scanning"],
						orgLevelRuleset.CreatedAt,
						orgLevelRuleset.UpdatedAt,
					})
					if err != nil {
						zap.S().Error("Error raised in writing output", zap.Error(err))
						return err
					}

				}
			}
			zap.S().Infof("Successfully listed organization level rulesets for %s", owner)
		}

		if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
			zap.S().Infof("Gathering repositories specified in org %s to list rulesets for", owner)
			allRepos, err := g.GatherRepositories(owner, repos)
			if err != nil {
				zap.S().Error("Error raised in gathering repos", zap.Error(err))
				return err
			}
			allRepoRules, err := g.FetchRepoRulesets(owner, allRepos)
			if err != nil {
				zap.S().Error("Error raised in fetching repo ruleset data", zap.Error(err))
				return err
			}
			for _, singleRepoRule := range allRepoRules {
				zap.S().Infof("Gathering specific ruleset data for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)
				repoLevelRulesetResponse, err := g.GetRepoLevelRuleset(owner, singleRepoRule.RepoName, singleRepoRule.Rule.DatabaseID)
				if err != nil {
					zap.S().Error("Error raised in getting repo variables", zap.Error(err))
					continue
				}
				var repoLevelRuleset data.RepoRuleset
				err = json.Unmarshal(repoLevelRulesetResponse, &repoLevelRuleset)
				if err != nil {
					zap.S().Error("Error raised with variable response", zap.Error(err))
					continue
				}
				Actors := g.ProcessActorsForExport(repoLevelRuleset.BypassActors, owner, orgID, singleRepoRule.Rule.ID)
				repoRulesMap := g.ProcessRules(repoLevelRuleset.Rules)

				repoConditions := utils.ProcessConditions(repoLevelRuleset)

				zap.S().Debugf("Writing output for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)

				err = csvWriter.Write([]string{
					repoLevelRuleset.SourceType,
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
					repoRulesMap["creation"],
					repoRulesMap["update"],
					repoRulesMap["deletion"],
					repoRulesMap["required_linear_history"],
					repoRulesMap["merge_queue"],
					repoRulesMap["required_deployments"],
					repoRulesMap["required_signatures"],
					repoRulesMap["pull_request"],
					repoRulesMap["required_status_checks"],
					repoRulesMap["non_fast_forward"],
					repoRulesMap["commit_message_pattern"],
					repoRulesMap["commit_author_email_pattern"],
					repoRulesMap["committer_email_pattern"],
					repoRulesMap["branch_name_pattern"],
					repoRulesMap["tag_name_pattern"],
					repoRulesMap["file_path_restriction"],
					repoRulesMap["max_file_path_length"],
					repoRulesMap["file_extension_restriction"],
					repoRulesMap["max_file_size"],
					repoRulesMap["workflows"],
					repoRulesMap["code_scanning"],
					repoLevelRuleset.CreatedAt,
					repoLevelRuleset.UpdatedAt,
				})
				if err != nil {
					zap.S().Error("Error raised in writing output", zap.Error(err))
					return err
				}
			}
			zap.S().Infof("Successfully listed repository level rulesets for %s", owner)
		}

		zap.S().Infof("Successfully listed all rulesets for %s", owner)
		csvWriter.Flush()
	}
	return nil
}
