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
			if cmdFlags.debug {
				logger, _ := log.NewLogger(cmdFlags.debug)
				defer logger.Sync() // nolint:errcheck
				zap.ReplaceGlobals(logger)
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
	// Configure flags for command

	listCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub Personal Access Token (default "gh auth token")`)
	listCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	listCmd.Flags().StringVarP(&cmdFlags.listFile, "output-file", "o", reportFileDefault, "Name of file to write CSV list to")
	listCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")

	listCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")

	return listCmd
}

func runCmdList(owner string, repos []string, cmdFlags *cmdFlags, g *utils.APIGetter, reportWriter io.Writer) error {
	fmt.Printf("Gathering repositories and/or rulesets for %s\n", owner)
	var reposCursor, repoRulesCursor, orgRulesCursor *string
	var allOrgRules []data.Rulesets
	var allRepoRules []data.RepoNameRule
	var allRepos []data.RepoInfo

	csvWriter := csv.NewWriter(reportWriter)

	err := csvWriter.Write([]string{
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
		for {
			orgRulesetsQuery, err := g.GetOrgRulesetsList(owner, orgRulesCursor)
			if err != nil {
				zap.S().Error("Error raised in getting organization ruleset list", zap.Error(err))
				return err
			}

			allOrgRules = append(allOrgRules, orgRulesetsQuery.Organization.Rulesets.Nodes...)
			orgRulesCursor = &orgRulesetsQuery.Organization.Rulesets.PageInfo.EndCursor
			if !orgRulesetsQuery.Organization.Rulesets.PageInfo.HasNextPage {
				break
			}

		}

		for _, singleRule := range allOrgRules {
			var Actors, PropertyInclude, PropertyExclude []string
			var includeNames, excludeNames, boolNames, includeRefNames, excludeRefNames string

			zap.S().Infof("Gathering specific ruleset data for org rule %s", singleRule.Name)
			orgLevelRulesetResponse, err := g.GetOrgLevelRuleset(owner, singleRule.DatabaseID)
			if err != nil {
				zap.S().Error("Error raised in getting org level ruleset data for %s", singleRule.DatabaseID, zap.Error(err))
				continue
			} else {
				var orgLevelRuleset data.RepoRuleset
				err = json.Unmarshal(orgLevelRulesetResponse, &orgLevelRuleset)
				if err != nil {
					zap.S().Error("Error raised with variable response", zap.Error(err))
					continue
				}

				Actors = utils.ProcessActors(orgLevelRuleset.BypassActors)
				if orgLevelRuleset.Conditions != nil {
					if orgLevelRuleset.Conditions.RepositoryProperty != nil {
						PropertyInclude = utils.ProcessProperties(orgLevelRuleset.Conditions.RepositoryProperty.Include)
						PropertyExclude = utils.ProcessProperties(orgLevelRuleset.Conditions.RepositoryProperty.Exclude)
					}
					if orgLevelRuleset.Conditions.RepositoryName != nil {
						includeNames = strings.Join(orgLevelRuleset.Conditions.RepositoryName.Include, ";")
						excludeNames = strings.Join(orgLevelRuleset.Conditions.RepositoryName.Exclude, ";")
						boolNames = strconv.FormatBool(orgLevelRuleset.Conditions.RepositoryName.Protected)
					}
					includeRefNames = strings.Join(orgLevelRuleset.Conditions.RefName.Include, ";")
					excludeRefNames = strings.Join(orgLevelRuleset.Conditions.RefName.Exclude, ";")
				}
				rulesMap := utils.ProcessRules(orgLevelRuleset.Rules)

				zap.S().Debugf("Writing output for org rule %s", singleRule.Name)

				err = csvWriter.Write([]string{
					orgLevelRuleset.SourceType,
					"N/A",
					strconv.Itoa(orgLevelRuleset.ID),
					orgLevelRuleset.Name,
					orgLevelRuleset.Target,
					orgLevelRuleset.Enforcement,
					strings.Join(Actors, "|"),
					includeRefNames,
					excludeRefNames,
					includeNames,
					excludeNames,
					boolNames,
					strings.Join(PropertyInclude, "|"),
					strings.Join(PropertyExclude, "|"),
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
		fmt.Printf("Successfully listed organization level rulesets for %s\n", owner)
	}

	if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
		zap.S().Infof("Gathering repositories specified in org %s to list rulesets for", owner)
		if len(repos) > 0 {
			zap.S().Infof("Processing repos: %s", repos)
			for _, repo := range repos {
				zap.S().Debugf("Processing %s/%s", owner, repo)
				repoQuery, err := g.GetRepo(owner, repo)
				if err != nil {
					zap.S().Error("Error raised in getting repo", repo, zap.Error(err))
					continue
				} else {
					allRepos = append(allRepos, repoQuery.Repository)
				}
			}
		} else {
			// Prepare writer for outputting report
			for {
				zap.S().Debugf("Processing list of repositories for %s", owner)
				reposQuery, err := g.GetReposList(owner, reposCursor)
				if err != nil {
					zap.S().Error("Error raised in processing list of repos", zap.Error(err))
					return err
				}
				allRepos = append(allRepos, reposQuery.Organization.Repositories.Nodes...)
				reposCursor = &reposQuery.Organization.Repositories.PageInfo.EndCursor
				if !reposQuery.Organization.Repositories.PageInfo.HasNextPage {
					break
				}
			}
		}
		for _, repo := range allRepos {
			zap.S().Infof("Checking for rulesets in repo %s", repo.Name)
			for {
				repoRulesetsQuery, err := g.GetRepoRulesetsList(owner, repo.Name, repoRulesCursor)
				if err != nil {
					return err
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
		for _, singleRepoRule := range allRepoRules {
			var Actors, PropertyInclude, PropertyExclude []string
			var includeNames, excludeNames, boolNames, includeRefNames, excludeRefNames string
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
			Actors = utils.ProcessActors(repoLevelRuleset.BypassActors)
			repoRulesMap := utils.ProcessRules(repoLevelRuleset.Rules)

			if repoLevelRuleset.Conditions != nil {
				if repoLevelRuleset.Conditions.RepositoryName != nil {
					includeNames = strings.Join(repoLevelRuleset.Conditions.RepositoryName.Include, ";")
					excludeNames = strings.Join(repoLevelRuleset.Conditions.RepositoryName.Exclude, ";")
					boolNames = strconv.FormatBool(repoLevelRuleset.Conditions.RepositoryName.Protected)
				}
				if repoLevelRuleset.Conditions.RepositoryProperty != nil {
					PropertyInclude = utils.ProcessProperties(repoLevelRuleset.Conditions.RepositoryProperty.Include)
					PropertyExclude = utils.ProcessProperties(repoLevelRuleset.Conditions.RepositoryProperty.Exclude)
				}
				includeRefNames = strings.Join(repoLevelRuleset.Conditions.RefName.Include, ";")
				excludeRefNames = strings.Join(repoLevelRuleset.Conditions.RefName.Exclude, ";")
			}

			zap.S().Debugf("Writing output for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)

			err = csvWriter.Write([]string{
				repoLevelRuleset.SourceType,
				singleRepoRule.RepoName,
				strconv.Itoa(repoLevelRuleset.ID),
				repoLevelRuleset.Name,
				repoLevelRuleset.Target,
				repoLevelRuleset.Enforcement,
				strings.Join(Actors, "|"),
				includeRefNames,
				excludeRefNames,
				includeNames,
				excludeNames,
				boolNames,
				strings.Join(PropertyInclude, "|"),
				strings.Join(PropertyExclude, "|"),
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
	}

	fmt.Printf("Successfully listed repository level rulesets for %s\n", owner)
	csvWriter.Flush()

	return nil
}
