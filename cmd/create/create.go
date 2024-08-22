package create

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/katiem0/gh-migrate-rulesets/internal/log"
	"github.com/katiem0/gh-migrate-rulesets/internal/utils"

	// Add this line
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type cmdFlags struct {
	sourceToken    string
	sourceOrg      string
	sourceHostname string
	token          string
	hostname       string
	fileName       string
	ruleType       string
	debug          bool
}

func NewCmdCreate() *cobra.Command {
	cmdFlags := cmdFlags{}
	var authToken, authSourceToken string

	createCmd := &cobra.Command{
		Use:   "create [flags] <organization> [repo ...]",
		Short: "Create repository rulesets",
		Long:  "Create repository rulesets at the repo and/or org level from a file.",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(createCmd *cobra.Command, args []string) error {
			if len(cmdFlags.fileName) == 0 && len(cmdFlags.sourceOrg) == 0 {
				return errors.New("a file or source organization must be specified where rulesets will be created from")
			} else if len(cmdFlags.sourceOrg) > 0 && len(cmdFlags.sourceToken) == 0 {
				return errors.New("a personal access token must be specified to access rulesets from the Source Organization")
			} else if len(cmdFlags.fileName) > 0 && len(cmdFlags.sourceOrg) > 0 {
				return errors.New("specify only one of `--source-organization` or `from-file`")
			}
			return nil
		},
		RunE: func(createCmd *cobra.Command, args []string) error {
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

			authSourceToken = utils.GetAuthToken(cmdFlags.sourceToken, cmdFlags.sourceHostname)
			restSrcClient, gqlSrcClient, err := utils.InitializeClients(cmdFlags.sourceHostname, authSourceToken)
			if err != nil {
				return err
			}
			owner := args[0]
			repos := args[1:]

			return runCmdCreate(owner, repos, &cmdFlags, utils.NewAPIGetter(gqlClient, restClient), utils.NewAPIGetter(gqlSrcClient, restSrcClient))
		},
	}
	ruleDefault := "all"

	// Configure flags for command
	createCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub personal access token for organization to write to (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceToken, "source-token", "st", "", `GitHub personal access token for Source Organization (Required for --source-organization)`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceOrg, "source-organization", "so", "", `Name of the Source Organization to copy rulesets from (Requires --source-token)`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceHostname, "source-hostname", "", "github.com", "GitHub Enterprise Server hostname where rulesets are copied from")
	createCmd.Flags().StringVarP(&cmdFlags.fileName, "from-file", "f", "", "Path and Name of CSV file to create rulesets from")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")
	createCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")

	return createCmd
}

func runCmdCreate(owner string, repos []string, cmdFlags *cmdFlags, g *utils.APIGetter, s *utils.APIGetter) error {
	var rulesetData [][]string
	sourceOrg := cmdFlags.sourceOrg
	var importRepoRulesetsList []data.RepoRuleset
	zap.S().Infof("Reading in file %s to identify repository rulesets", cmdFlags.fileName)
	if len(cmdFlags.fileName) > 0 {
		f, err := os.Open(cmdFlags.fileName)
		zap.S().Debugf("Opening up file %s", cmdFlags.fileName)
		if err != nil {
			zap.S().Errorf("Error arose opening branch protection policies csv file")
			return err
		}
		defer f.Close()
		csvReader := csv.NewReader(f)
		rulesetData, err = csvReader.ReadAll()
		zap.S().Debugf("Reading in all lines from csv file")
		if err != nil {
			zap.S().Errorf("Error arose reading assignments from csv file")
			return err
		}
		importRepoRulesetsList = utils.CreateRepoRulesetsData(owner, rulesetData)
		for _, ruleset := range importRepoRulesetsList {
			createRuleset, err := utils.ProcessRulesets(ruleset)
			if err != nil {
				zap.S().Errorf("Error creating ruleset rules data: %v", err)
				continue
			}
			if createRuleset.Target == "push" {
				createRuleset.Conditions = nil
			} else {
				createRuleset.Conditions = utils.CleanConditions(createRuleset.Conditions)
			}

			createRulesetJSON, err := json.Marshal(createRuleset)
			if err != nil {
				zap.S().Errorf("Error marshaling ruleset: %v", err)
				continue
			}
			reader := bytes.NewReader(createRulesetJSON)
			if ruleset.SourceType == "Organization" {
				zap.S().Debugf("Creating rulesets under %s", owner)
				err = g.CreateOrgLevelRuleset(owner, reader)
				if err != nil {
					zap.S().Errorf("Error arose creating ruleset %s for %s", ruleset.Name, owner)
					continue
				}
				fmt.Printf("Successfully create repository ruleset %s for %s\n", ruleset.Name, owner)
			} else if ruleset.SourceType == "Repository" {
				zap.S().Debugf("Creating rulesets under %s", ruleset.Source)
				err = g.CreateRepoLevelRuleset(ruleset.Source, reader)
				if err != nil {
					zap.S().Errorf("Error arose creating ruleset %s for %s", ruleset.Name, ruleset.Source)
					continue
				}
				fmt.Printf("Successfully create repository ruleset %s for %s\n", ruleset.Name, ruleset.Source)
			}
		}
	} else if len(sourceOrg) > 0 {
		zap.S().Debugf("Reading in rulesets from source organization %s", sourceOrg)

		if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "orgOnly" {
			zap.S().Infof("Gathering source organization %s level rulesets", sourceOrg)
			allOrgRules, err := s.FetchOrgRulesets(sourceOrg)
			if err != nil {
				zap.S().Error("Error raised in fetching org  ruleset data for %s", sourceOrg, zap.Error(err))
			}
			for _, singleRule := range allOrgRules {
				zap.S().Infof("Gathering specific ruleset data for org rule %s", singleRule.Name)
				orgLevelRulesetResponse, err := s.GetOrgLevelRuleset(sourceOrg, singleRule.DatabaseID)
				if err != nil {
					zap.S().Error("Error raised in getting org level ruleset data for %s", singleRule.DatabaseID, zap.Error(err))
					continue
				} else {
					var orgLevelRuleset data.RepoRuleset
					err = json.Unmarshal(orgLevelRulesetResponse, &orgLevelRuleset)
					if utils.LogErrorAndContinue(err, "Error raised with variable response") {
						continue
					}
					createRuleset, err := utils.ProcessRulesets(orgLevelRuleset)
					if utils.LogErrorAndContinue(err, "Error creating rulesets data") {
						continue
					}
					createRulesetJSON, err := json.Marshal(createRuleset)
					if utils.LogErrorAndContinue(err, "Error marshaling ruleset:") {
						continue
					}
					reader := bytes.NewReader(createRulesetJSON)
					zap.S().Debugf("Creating rulesets under target organization %s", owner)
					err = s.CreateOrgLevelRuleset(owner, reader)
					if err != nil {
						zap.S().Errorf("Error arose creating ruleset %s for %s", orgLevelRuleset.Name, owner)
						continue
					}
				}
			}
		} else if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
			zap.S().Infof("Gathering repositories specified in org %s to list rulesets for", sourceOrg)
			allRepos, err := g.GatherRepositories(sourceOrg, repos)
			if err != nil {
				zap.S().Error("Error raised in gathering repos", zap.Error(err))
				return err
			}
			allRepoRules, err := g.FetchRepoRulesets(sourceOrg, allRepos)
			if err != nil {
				zap.S().Error("Error raised in fetching repo ruleset data", zap.Error(err))
				return err
			}
			for _, singleRepoRule := range allRepoRules {
				zap.S().Infof("Gathering specific ruleset data for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)
				repoLevelRulesetResponse, err := g.GetRepoLevelRuleset(sourceOrg, singleRepoRule.RepoName, singleRepoRule.Rule.DatabaseID)
				if err != nil {
					zap.S().Error("Error raised in getting repo variables", zap.Error(err))
					continue
				} else {
					var repoLevelRuleset data.RepoRuleset
					err = json.Unmarshal(repoLevelRulesetResponse, &repoLevelRuleset)
					if utils.LogErrorAndContinue(err, "Error raised with variable response") {
						continue
					}
					createRuleset, err := utils.ProcessRulesets(repoLevelRuleset)
					if utils.LogErrorAndContinue(err, "Error creating rulesets data") {
						continue
					}
					createRulesetJSON, err := json.Marshal(createRuleset)
					if utils.LogErrorAndContinue(err, "Error marshaling ruleset:") {
						continue
					}
					reader := bytes.NewReader(createRulesetJSON)
					repoName := strings.Split(repoLevelRuleset.Source, "/")[1]
					zap.S().Debugf("Creating rulesets under %s/%s", owner, repoName)
					newSource := fmt.Sprintf("%s/%s", owner, repoName)
					err = g.CreateRepoLevelRuleset(newSource, reader)
					if err != nil {
						zap.S().Errorf("Error arose creating ruleset %s for %s", repoLevelRuleset.Name, newSource)
						continue
					}
				}
			}
		} else {
			zap.S().Infof("Invalid ruleType specified: %s", cmdFlags.ruleType)
		}
	} else {
		zap.S().Errorf("Error arose identifying rulesets")
	}
	fmt.Printf("Successfully completed creating repository rulesets from %s in org %s", cmdFlags.fileName, owner)
	return nil
}
