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
	actorMapping   string
	repos          []string
	targetRepo     string
	ruleType       string
	dryRun         bool
	debug          bool
}

func NewCmdCreate() *cobra.Command {
	cmdFlags := cmdFlags{}
	var authToken, authSourceToken string

	createCmd := &cobra.Command{
		Use:   "create [flags] <organization>",
		Short: "Create repository rulesets",
		Long:  "Create repository rulesets at the repo and/or org level from a file or list.",
		Args:  validateOrgArg,
		PreRunE: func(createCmd *cobra.Command, args []string) error {
			if len(cmdFlags.fileName) == 0 && len(cmdFlags.sourceOrg) == 0 {
				return errors.New("a file or source organization must be specified where rulesets will be created from")
			} else if len(cmdFlags.fileName) > 0 && len(cmdFlags.sourceOrg) > 0 {
				return errors.New("specify only one of `--source-org` or `--from-file`")
			}
			if len(cmdFlags.targetRepo) > 0 {
				if len(cmdFlags.fileName) > 0 {
					return errors.New("`--target-repo` cannot be used with `--from-file`; the file already contains the destination repository name")
				}
				if len(cmdFlags.repos) != 1 {
					return errors.New("`--target-repo` requires exactly one repository via `--repos`")
				}
			}
			if len(cmdFlags.actorMapping) > 0 {
				if len(cmdFlags.fileName) > 0 {
					return errors.New("`--actor-mapping` cannot be used with `--from-file`; update the target IDs directly in the file")
				}
				if _, err := os.Stat(cmdFlags.actorMapping); err != nil {
					return fmt.Errorf("actor mapping file not found: %w", err)
				}
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

	createCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub personal access token for organization to write to (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceToken, "source-pat", "p", "", `GitHub personal access token for Source Organization (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceOrg, "source-org", "s", "", `Name of the Source Organization to copy rulesets from`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceHostname, "source-hostname", "", "github.com", "GitHub Enterprise Server hostname where rulesets are copied from")
	createCmd.Flags().StringVarP(&cmdFlags.fileName, "from-file", "f", "", "Path and Name of CSV file to create rulesets from")
	createCmd.Flags().StringVarP(&cmdFlags.actorMapping, "actor-mapping", "", "", "Path and Name of CSV file mapping source bypass actor IDs to target IDs (for predefined repository roles and renamed actors)")
	createCmd.Flags().StringSliceVarP(&cmdFlags.repos, "repos", "R", []string{}, "List of repositories names to recreate rulesets for separated by commas (i.e. repo1,repo2,repo3)")
	createCmd.Flags().StringVarP(&cmdFlags.targetRepo, "target-repo", "T", "", "Rename the destination repository when migrating a single repository's rulesets")
	createCmd.PersistentFlags().StringVarP(&cmdFlags.ruleType, "ruleType", "r", ruleDefault, "List rulesets for a specific application or all: {all|repoOnly|orgOnly}")
	createCmd.Flags().BoolVarP(&cmdFlags.dryRun, "dry-run", "", false, "Preview ruleset creates without writing changes")
	createCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")

	return createCmd
}

func runCmdCreate(owner string, cmdFlags *cmdFlags, g utils.Getter, s utils.Getter) error {
	repos := cmdFlags.repos
	var errorMessage string
	var rulesetData [][]string
	sourceOrg := cmdFlags.sourceOrg
	var sourceOrgID int
	var importRepoRulesetsList []data.RepoRuleset
	var errorRulesets []data.ErrorRulesets
	successCount := 0

	zap.S().Infof("Reading in file %s to identify repository rulesets", cmdFlags.fileName)
	if len(cmdFlags.fileName) > 0 {
		f, err := os.Open(cmdFlags.fileName)
		zap.S().Debugf("Opening up file %s", cmdFlags.fileName)
		if err != nil {
			zap.S().Errorf("Error arose opening branch protection policies csv file")
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				zap.S().Errorf("Error closing file: %v", err)
			}
		}()
		csvReader := csv.NewReader(f)
		rulesetData, err = csvReader.ReadAll()
		zap.S().Debugf("Reading in all lines from csv file")
		if err != nil {
			zap.S().Errorf("Error arose reading assignments from csv file")
			return err
		}
		// From-file rulesets already carry target IDs (including bypass actors);
		// mapping and live ID translation apply only to the --source-org path.
		importRepoRulesetsList = g.CreateRepoRulesetsData(owner, rulesetData, nil)
		for _, ruleset := range importRepoRulesetsList {
			execErr := utils.SafeExecute(func() error {
				createRuleset, err := utils.ProcessRulesets(ruleset)
				if err != nil {
					zap.S().Errorf("Error creating ruleset rules data: %v", err)
					errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: ruleset.Source, RulesetName: ruleset.Name, Error: err.Error()})
					return nil
				}
				if createRuleset.Target == "push" {
					createRuleset.Conditions = nil
				} else {
					createRuleset.Conditions = utils.CleanConditions(createRuleset.Conditions)
				}
				createRulesetJSON, err := json.Marshal(createRuleset)
				if err != nil {
					zap.S().Errorf("Error marshaling ruleset: %v", err)
					errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: ruleset.Source, RulesetName: createRuleset.Name, Error: err.Error()})
					return nil
				}
				reader := bytes.NewReader(createRulesetJSON)
				switch ruleset.SourceType {
				case "Organization":
					if cmdFlags.dryRun {
						zap.S().Infof("[dry-run] would CREATE org ruleset %s under %s", createRuleset.Name, owner)
						successCount++
						return nil
					}
					zap.S().Debugf("Creating organization rulesets under %s", owner)
					err = g.CreateOrgLevelRuleset(owner, reader)
					if err != nil {
						errorMessage = extractErrorMessage(err)
						errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: owner, RulesetName: createRuleset.Name, Error: errorMessage})
						zap.S().Infof("Error creating ruleset %s for %s: %s", ruleset.Source, createRuleset.Name, errorMessage)
						return nil
					}
					zap.S().Infof("Successfully create repository ruleset %s for %s", ruleset.Name, owner)
					successCount++
				case "Repository":
					destSource := ruleset.Source
					if ruleset.TargetSource != "" {
						destSource = ruleset.TargetSource
					}
					if destSource != ruleset.Source {
						zap.S().Infof("Migrating rulesets from %s to %s", ruleset.Source, destSource)
					}
					zap.S().Debugf("Trying to create repository rulesets under %s", destSource)
					exists := g.RepoExists(destSource)
					if !exists {
						zap.S().Debugf("Repository %s does not exist", destSource)
						errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: destSource, RulesetName: createRuleset.Name, Error: "Repository does not exist"})
						zap.S().Infof("Error creating ruleset %s for %s (source repo %s): %s", createRuleset.Name, destSource, ruleset.Source, "Repository does not exist")
						return nil
					}
					zap.S().Debugf("Creating rulesets under %s", destSource)
					if cmdFlags.dryRun {
						zap.S().Infof("[dry-run] would CREATE repo ruleset %s under %s", createRuleset.Name, destSource)
						successCount++
						return nil
					}
					err = g.CreateRepoLevelRuleset(destSource, reader)
					if err != nil {
						errorMessage = extractErrorMessage(err)
						errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: destSource, RulesetName: createRuleset.Name, Error: errorMessage})
						zap.S().Infof("Error creating ruleset %s for %s (source repo %s): %s", createRuleset.Name, destSource, ruleset.Source, errorMessage)
						return nil
					}
					zap.S().Infof("Successfully create repository ruleset %s for %s", ruleset.Name, destSource)
					successCount++
				}
				return nil
			}, fmt.Sprintf("creating ruleset %s for %s", ruleset.Name, ruleset.Source))
			if execErr != nil {
				errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: ruleset.Source, RulesetName: ruleset.Name, Error: execErr.Error()})
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

			actorMapping, err := utils.LoadActorMapping(cmdFlags.actorMapping)
			if err != nil {
				return err
			}

			zap.S().Infoln("Reading in rulesets from source organization", sourceOrg)

			if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "orgOnly" {
				zap.S().Infof("Gathering source organization %s level rulesets", sourceOrg)
				allOrgRules, err := s.FetchOrgRulesets(sourceOrg)
				if err != nil {
					zap.S().Errorf("Error raised in fetching org ruleset data for %s", sourceOrg)
					errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: "N/A", Error: fmt.Sprintf("failed to fetch organization rulesets: %v", err)})
				}
				for _, singleRule := range allOrgRules {
					execErr := utils.SafeExecute(func() error {
						zap.S().Debugf("Gathering specific ruleset data for org rule %s", singleRule.Name)
						orgLevelRulesetResponse, err := s.GetOrgLevelRuleset(sourceOrg, singleRule.DatabaseID)
						if err != nil {
							zap.S().Errorf("Error raised in getting org level ruleset data for %d: %v", singleRule.DatabaseID, err)
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: singleRule.Name, Error: err.Error()})
							return nil
						}
						var orgLevelRuleset data.RepoRuleset
						err = json.Unmarshal(orgLevelRulesetResponse, &orgLevelRuleset)
						if err != nil {
							zap.S().Error("Error raised with variable response", zap.Error(err))
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: singleRule.Name, Error: err.Error()})
							return nil
						}
						translatedOrgRuleset, tErr := utils.TranslateRuleset(g, s, owner, sourceOrg, sourceOrgID, orgLevelRuleset, actorMapping)
						if tErr != nil {
							recordTranslationSkip(tErr, sourceOrg, singleRule.Name, &errorRulesets)
							return nil
						}
						createRuleset, err := utils.ProcessRulesets(translatedOrgRuleset)
						if err != nil {
							zap.S().Errorf("Error creating rulesets data: %v", err)
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: singleRule.Name, Error: err.Error()})
							return nil
						}
						createRulesetJSON, err := json.Marshal(createRuleset)
						if err != nil {
							zap.S().Errorf("Error marshaling ruleset: %v", err)
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: createRuleset.Name, Error: err.Error()})
							return nil
						}
						reader := bytes.NewReader(createRulesetJSON)
						if cmdFlags.dryRun {
							zap.S().Infof("[dry-run] would CREATE org ruleset %s under %s", createRuleset.Name, owner)
							successCount++
							return nil
						}
						zap.S().Debugf("Creating rulesets under target organization %s", owner)
						err = g.CreateOrgLevelRuleset(owner, reader)
						if err != nil {
							errorMessage = extractErrorMessage(err)
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: createRuleset.Name, Error: errorMessage})
							zap.S().Infof("Error creating ruleset %s for %s: %s", sourceOrg, createRuleset.Name, errorMessage)
							return nil
						}
						zap.S().Infof("Successfully created organization ruleset %s for %s", createRuleset.Name, owner)
						successCount++
						return nil
					}, fmt.Sprintf("creating org ruleset %s", singleRule.Name))
					if execErr != nil {
						errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: singleRule.Name, Error: execErr.Error()})
					}
				}
			}

			if cmdFlags.ruleType == "all" || cmdFlags.ruleType == "repoOnly" {
				zap.S().Infof("Gathering repositories specified in org %s to list rulesets for", sourceOrg)
				allRepos := s.GatherRepositories(sourceOrg, repos)

				if len(allRepos) > 0 {
					allRepoRules, err := s.FetchRepoRulesets(sourceOrg, allRepos)
					if err != nil {
						zap.S().Error("Error raised in fetching repo ruleset data", zap.Error(err))
						errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: sourceOrg, RulesetName: "N/A", Error: fmt.Sprintf("failed to fetch repository rulesets: %v", err)})
						allRepoRules = nil
					}
					for _, singleRepoRule := range allRepoRules {
						execErr := utils.SafeExecute(func() error {
							zap.S().Debugf("Gathering specific ruleset data for repo %s rule %s", singleRepoRule.RepoName, singleRepoRule.Rule.Name)
							repoLevelRulesetResponse, err := s.GetRepoLevelRuleset(sourceOrg, singleRepoRule.RepoName, singleRepoRule.Rule.DatabaseID)
							if err != nil {
								zap.S().Error("Error raised in getting repo variables", zap.Error(err))
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: singleRepoRule.RepoName, RulesetName: singleRepoRule.Rule.Name, Error: err.Error()})
								return nil
							}
							var repoLevelRuleset data.RepoRuleset
							err = json.Unmarshal(repoLevelRulesetResponse, &repoLevelRuleset)
							if err != nil {
								zap.S().Debugf("Error raised with variable response")
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: singleRepoRule.RepoName, RulesetName: singleRepoRule.Rule.Name, Error: err.Error()})
								return nil
							}
							translatedRepoRuleset, tErr := utils.TranslateRuleset(g, s, owner, sourceOrg, sourceOrgID, repoLevelRuleset, actorMapping)
							if tErr != nil {
								recordTranslationSkip(tErr, singleRepoRule.RepoName, singleRepoRule.Rule.Name, &errorRulesets)
								return nil
							}
							createRuleset, err := utils.ProcessRulesets(translatedRepoRuleset)
							if err != nil {
								zap.S().Errorf("Error creating rulesets data: %v", err)
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: repoLevelRuleset.Source, RulesetName: singleRepoRule.Rule.Name, Error: err.Error()})
								return nil
							}
							createRulesetJSON, err := json.Marshal(createRuleset)
							if err != nil {
								zap.S().Errorf("Error marshaling ruleset: %v", err)
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: repoLevelRuleset.Source, RulesetName: createRuleset.Name, Error: err.Error()})
								return nil
							}
							reader := bytes.NewReader(createRulesetJSON)

							sourceParts := strings.Split(repoLevelRuleset.Source, "/")
							if len(sourceParts) < 2 {
								zap.S().Errorf("Unexpected ruleset source format %q", repoLevelRuleset.Source)
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: repoLevelRuleset.Source, RulesetName: createRuleset.Name, Error: "Unexpected ruleset source format"})
								return nil
							}
							repoName := sourceParts[1]
							if len(cmdFlags.targetRepo) > 0 {
								zap.S().Infof("Migrating rulesets from %s to %s", repoName, cmdFlags.targetRepo)
								repoName = cmdFlags.targetRepo
							}
							zap.S().Debugf("Creating rulesets under %s/%s", owner, repoName)
							newSource := fmt.Sprintf("%s/%s", owner, repoName)
							exists := g.RepoExists(newSource)
							if !exists {
								zap.S().Debugf("Repository %s does not exist in %s", repoName, owner)
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: newSource, RulesetName: createRuleset.Name, Error: "Repository does not exist"})
								zap.S().Infof("Error creating ruleset %s for %s (source repo %s): %s", createRuleset.Name, newSource, repoLevelRuleset.Source, "Repository does not exist")
								return nil
							}
							if cmdFlags.dryRun {
								zap.S().Infof("[dry-run] would CREATE repo ruleset %s under %s", createRuleset.Name, newSource)
								successCount++
								return nil
							}
							err = g.CreateRepoLevelRuleset(newSource, reader)
							if err != nil {
								errorMessage = extractErrorMessage(err)
								errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: newSource, RulesetName: createRuleset.Name, Error: errorMessage})
								zap.S().Infof("Error creating ruleset %s for %s (source repo %s): %s", createRuleset.Name, newSource, repoLevelRuleset.Source, errorMessage)
								return nil
							}
							zap.S().Infof("Successfully created repository ruleset %s for %s", singleRepoRule.Rule.Name, newSource)
							successCount++
							return nil
						}, fmt.Sprintf("creating repo ruleset %s for %s", singleRepoRule.Rule.Name, singleRepoRule.RepoName))
						if execErr != nil {
							errorRulesets = append(errorRulesets, data.ErrorRulesets{Source: singleRepoRule.RepoName, RulesetName: singleRepoRule.Rule.Name, Error: execErr.Error()})
						}
					}
				} else {
					zap.S().Warnf("No repositories found in %s. Skipping repository-level rulesets.", sourceOrg)
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
		} else {
			zap.S().Warnf("%d ruleset(s) failed to create. Failures written to %s", len(errorRulesets), reportFileName)
		}
	}
	if len(cmdFlags.fileName) > 0 {
		zap.S().Infof("Completed list of rulesets from %s in org %s", cmdFlags.fileName, owner)
	} else {
		zap.S().Infof("Completed list of rulesets in org %s", owner)
	}
	if cmdFlags.dryRun {
		zap.S().Infof("Summary: %d ruleset(s) would be created (dry run), %d skipped/failed", successCount, len(errorRulesets))
	} else {
		zap.S().Infof("Summary: %d ruleset(s) created successfully, %d failed", successCount, len(errorRulesets))
	}
	return nil
}

// validateOrgArg enforces the single <organization> positional. Extra positionals
// usually mean an empty variable shifted a flag's value, so the error says so.
func validateOrgArg(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 1:
		return nil
	case 0:
		return errors.New("an organization argument is required")
	default:
		return fmt.Errorf("expected a single organization argument but received %d (%s); check for an unset token or variables", len(args), strings.Join(args, ", "))
	}
}

func recordTranslationSkip(err error, source, rulesetName string, errorRulesets *[]data.ErrorRulesets) {
	*errorRulesets = append(*errorRulesets, data.ErrorRulesets{Source: source, RulesetName: rulesetName, Error: err.Error()})
	zap.S().Infof("Skipping ruleset %s for %s: %s", rulesetName, source, errorExcerpt(err, 2))
}

// errorExcerpt flattens a joined error onto one log line, keeping the first few
// causes; the full set is still written to the error output file.
func errorExcerpt(err error, keep int) string {
	causes := strings.Split(err.Error(), "\n")
	if len(causes) <= keep {
		return strings.Join(causes, "; ")
	}
	return fmt.Sprintf("%s; (+%d more)", strings.Join(causes[:keep], "; "), len(causes)-keep)
}

// Drops go-gh's "HTTP 422: ... (<url>)" status line; the reasons after it are
// newline-joined, so they are flattened onto one line.
func extractErrorMessage(err error) string {
	if _, detail, found := strings.Cut(err.Error(), "\n"); found {
		return strings.ReplaceAll(detail, "\n", "; ")
	}
	return err.Error()
}
