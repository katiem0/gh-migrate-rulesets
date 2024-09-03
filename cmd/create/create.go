package create

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	repos          []string
	ruleType       string
	debug          bool
}

func NewCmdCreate() *cobra.Command {
	cmdFlags := cmdFlags{}
	var authToken, authSourceToken string

	createCmd := &cobra.Command{
		Use:   "create [flags] <organization>",
		Short: "Create repository rulesets",
		Long:  "Create repository rulesets at the repo and/or org level from a file or list.",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(createCmd *cobra.Command, args []string) error {
			if len(cmdFlags.fileName) == 0 && len(cmdFlags.sourceOrg) == 0 {
				return errors.New("a file or source organization must be specified where rulesets will be created from")
			} else if len(cmdFlags.fileName) > 0 && len(cmdFlags.sourceOrg) > 0 {
				return errors.New("specify only one of `--source-organization` or `from-file`")
			}
			return nil
		},
		RunE: func(createCmd *cobra.Command, args []string) error {
			logger, _ := log.NewLogger(cmdFlags.debug)
			defer logger.Sync() // nolint:errcheck
			zap.ReplaceGlobals(logger)

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

			return runCmdCreate(owner, &cmdFlags, utils.NewAPIGetter(gqlClient, restClient), utils.NewAPIGetter(gqlSrcClient, restSrcClient))
		},
	}
	ruleDefault := "all"

	// Configure flags for command
	createCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub personal access token for organization to write to (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceToken, "source-pat", "p", "", `GitHub personal access token for Source Organization (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceOrg, "source-org", "s", "", `Name of the Source Organization to copy rulesets from`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceHostname, "source-hostname", "", "github.com", "GitHub Enterprise Server hostname where rulesets are copied from")
	createCmd.Flags().StringVarP(&cmdFlags.fileName, "from-file", "f", "", "Path and Name of CSV file to create rulesets from")
	createCmd.Flags().StringSliceVarP(&cmdFlags.repos, "repos", "R", []string{}, "List of repositories names to recreate rulesets for separated by commas (i.e. repo1,repo2,repo3)")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")
	createCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")

	return createCmd
}

func runCmdCreate(owner string, cmdFlags *cmdFlags, g *utils.APIGetter, s *utils.APIGetter) error {
	repos := cmdFlags.repos
	var errorValidation string
	var rulesetData [][]string
	sourceOrg := cmdFlags.sourceOrg
	var sourceOrgID int
	var importRepoRulesetsList []data.RepoRuleset
	var errorRulesets []data.ErrorRulesets

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
		importRepoRulesetsList = g.CreateRepoRulesetsData(owner, rulesetData)
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
					if strings.Contains(err.Error(), "\n") {
						errorValidation = strings.Split(err.Error(), "\n")[1]
					} else {
						errorValidation = err.Error()
					}
					errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: owner, RulesetName: createRuleset.Name, Error: errorValidation})
					zap.S().Infof("Error creating ruleset %s for %s: %s", ruleset.Source, createRuleset.Name, errorValidation)
					continue
				}
				zap.S().Infof("Successfully create repository ruleset %s for %s", ruleset.Name, owner)
			} else if ruleset.SourceType == "Repository" {
				exists := g.RepoExists(ruleset.Source)
				if !exists {
					zap.S().Debugf("Repository %s does not exist", ruleset.Source)
					errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: ruleset.Source, RulesetName: createRuleset.Name, Error: "Repository does not exist"})
					zap.S().Infof("Error creating ruleset %s for %s: %s", ruleset.Source, createRuleset.Name, "Repository does not exist")
					continue
				} else {
					{
						zap.S().Debugf("Creating rulesets under %s", ruleset.Source)
						err = g.CreateRepoLevelRuleset(ruleset.Source, reader)
						if err != nil {
							if strings.Contains(err.Error(), "\n") {
								errorValidation = strings.Split(err.Error(), "\n")[1]
							} else {
								errorValidation = err.Error()
							}
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: ruleset.Source, RulesetName: createRuleset.Name, Error: errorValidation})
							zap.S().Infof("Error creating ruleset %s for %s: %s", ruleset.Source, createRuleset.Name, errorValidation)

							continue
						}
					}
				}
				zap.S().Infof("Successfully create repository ruleset %s for %s", ruleset.Name, ruleset.Source)
			}
		}
	} else if len(sourceOrg) > 0 {
		zap.S().Debugln("Getting source organization ID")
		sourceOrgIDData, err := s.FetchOrgId(sourceOrg)
		if err != nil {
			zap.S().Error("Error raised in fetching org")
			return err
		} else {
			sourceOrgID = sourceOrgIDData.Organization.DatabaseID

			zap.S().Infoln("Reading in rulesets from source organization", sourceOrg)

			if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "orgOnly" {
				zap.S().Infof("Gathering source organization %s level rulesets", sourceOrg)
				allOrgRules, err := s.FetchOrgRulesets(sourceOrg)
				if err != nil {
					zap.S().Errorf("Error raised in fetching org ruleset data for %s", sourceOrg)
				}
				for _, singleRule := range allOrgRules {
					zap.S().Debugf("Gathering specific ruleset data for org rule %s", singleRule.Name)
					orgLevelRulesetResponse, err := s.GetOrgLevelRuleset(sourceOrg, singleRule.DatabaseID)
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
						updatedOrgLevelRuleset := g.UpdateBypassActorID(owner, sourceOrg, sourceOrgID, orgLevelRuleset, s)
						updatedOrgWorkflowRuleset := g.UpdateRequiredWorkflowRepoID(owner, updatedOrgLevelRuleset, s)
						createRuleset, err := utils.ProcessRulesets(updatedOrgWorkflowRuleset)
						if err != nil {
							zap.S().Errorf("Error creating rulesets data: %v", err)
							continue
						}
						createRulesetJSON, err := json.Marshal(createRuleset)
						if err != nil {
							zap.S().Errorf("Error marshaling ruleset: %v", err)
							continue
						}
						reader := bytes.NewReader(createRulesetJSON)
						zap.S().Debugf("Creating rulesets under target organization %s", owner)
						err = g.CreateOrgLevelRuleset(owner, reader)
						if err != nil {
							if strings.Contains(err.Error(), "\n") {
								errorValidation = strings.Split(err.Error(), "\n")[1]
							} else {
								errorValidation = err.Error()
							}
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: createRuleset.Name, Error: errorValidation})
							zap.S().Infof("Error creating ruleset %s for %s: %s", sourceOrg, createRuleset.Name, errorValidation)
							continue
						}
					}
				}
			}

			if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
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
					zap.S().Debugf("Gathering specific ruleset data for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)
					repoLevelRulesetResponse, err := g.GetRepoLevelRuleset(sourceOrg, singleRepoRule.RepoName, singleRepoRule.Rule.DatabaseID)
					if err != nil {
						zap.S().Error("Error raised in getting repo variables", zap.Error(err))
						continue
					} else {
						var repoLevelRuleset data.RepoRuleset
						err = json.Unmarshal(repoLevelRulesetResponse, &repoLevelRuleset)
						if err != nil {
							zap.S().Debugf("Error raised with variable response")
							continue
						}
						updatedRepoLevelRuleset := g.UpdateBypassActorID(owner, sourceOrg, sourceOrgID, repoLevelRuleset, s)
						updatedRepoWorkflowRuleset := g.UpdateRequiredWorkflowRepoID(owner, updatedRepoLevelRuleset, s)
						createRuleset, err := utils.ProcessRulesets(updatedRepoWorkflowRuleset)
						if err != nil {
							zap.S().Errorf("Error creating rulesets data: %v", err)
							continue
						}
						createRulesetJSON, err := json.Marshal(createRuleset)
						if err != nil {
							zap.S().Errorf("Error marshaling ruleset: %v", err)
							continue
						}
						reader := bytes.NewReader(createRulesetJSON)

						repoName := strings.Split(repoLevelRuleset.Source, "/")[1]
						zap.S().Debugf("Creating rulesets under %s/%s", owner, repoName)
						newSource := fmt.Sprintf("%s/%s", owner, repoName)
						exists := g.RepoExists(newSource)
						if !exists {
							zap.S().Debugf("Repository %s does not exist in %s", repoName, owner)
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: repoLevelRuleset.Source, RulesetName: createRuleset.Name, Error: "Repository does not exist"})
							zap.S().Infof("Error creating ruleset %s for %s: %s", repoLevelRuleset.Source, createRuleset.Name, "Repository does not exist")
							continue
						} else {
							err = g.CreateRepoLevelRuleset(newSource, reader)
							if err != nil {
								if strings.Contains(err.Error(), "\n") {
									errorValidation = strings.Split(err.Error(), "\n")[1]
								} else {
									errorValidation = err.Error()
								}
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: repoLevelRuleset.Source, RulesetName: createRuleset.Name, Error: errorValidation})
								zap.S().Infof("Error creating ruleset %s for %s: %s", repoLevelRuleset.Source, createRuleset.Name, errorValidation)
								continue
							}
						}

					}
				}
			}
		}
	} else {
		zap.S().Errorf("Error arose identifying rulesets")
	}

	if len(errorRulesets) > 0 {
		reportFileName := fmt.Sprintf("%s-ruleset-errors-%s.csv", owner, time.Now().Format("20060102150405"))
		err := utils.WriteErrorRulesetsToCSV(errorRulesets, reportFileName)
		if err != nil {
			zap.S().Errorf("Error writing error rulesets to csv file: %v", err)
		}
	}
	if len(cmdFlags.fileName) > 0 {
		zap.S().Infof("Completed list of rulesets from %s in org %s", cmdFlags.fileName, owner)
	} else {
		zap.S().Infof("Completed list of rulesets in org %s", owner)
	}
	return nil
}
