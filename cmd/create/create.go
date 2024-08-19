package create

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/katiem0/gh-migrate-rulesets/internal/log"
	"github.com/katiem0/gh-migrate-rulesets/internal/utils"

	// Add this line
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type cmdFlags struct {
	//sourceToken    string
	//sourceOrg      string
	//sourceHostname string
	token    string
	hostname string
	fileName string
	debug    bool
}

func NewCmdCreate() *cobra.Command {
	cmdFlags := cmdFlags{}
	var authToken string

	createCmd := &cobra.Command{
		Use:   "create [flags] <organization>",
		Short: "Create repository rulesets",
		Long:  "Create repository rulesets at the repo and/or org level from a file.",
		Args:  cobra.ExactArgs(1),
		// PreRunE: func(createCmd *cobra.Command, args []string) error {
		// 	if len(cmdFlags.fileName) == 0 && len(cmdFlags.sourceOrg) == 0 {
		// 		return errors.New("a file or source organization must be specified where rulesets will be created from")
		// 	} else if len(cmdFlags.sourceOrg) > 0 && len(cmdFlags.sourceToken) == 0 {
		// 		return errors.New("a personal access token must be specified to access rulesets from the Source Organization")
		// 	} else if len(cmdFlags.fileName) > 0 && len(cmdFlags.sourceOrg) > 0 {
		// 		return errors.New("specify only one of `--source-organization` or `from-file`")
		// 	}
		// 	return nil
		// },
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
			owner := args[0]

			return runCmdCreate(owner, &cmdFlags, utils.NewAPIGetter(gqlClient, restClient))
		},
	}
	// Configure flags for command	cmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub personal access token for organization to write to (default "gh auth token")`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.token, "token", "t", "", `GitHub personal access token for organization to write to (default "gh auth token")`)
	//createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceToken, "source-token", "s", "", `GitHub personal access token for Source Organization (Required for --source-organization)`)
	//createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceOrg, "source-organization", "o", "", `Name of the Source Organization to copy rulesets from (Requires --source-token)`)
	createCmd.PersistentFlags().StringVarP(&cmdFlags.hostname, "hostname", "", "github.com", "GitHub Enterprise Server hostname")
	//createCmd.PersistentFlags().StringVarP(&cmdFlags.sourceHostname, "source-hostname", "", "github.com", "GitHub Enterprise Server hostname where rulesets are copied from")
	createCmd.Flags().StringVarP(&cmdFlags.fileName, "from-file", "f", "", "Path and Name of CSV file to create repository rulesets from")
	createCmd.PersistentFlags().BoolVarP(&cmdFlags.debug, "debug", "d", false, "To debug logging")
	createCmd.MarkFlagRequired("from-file")

	return createCmd
}

func runCmdCreate(owner string, cmdFlags *cmdFlags, g *utils.APIGetter) error {
	var rulesetData [][]string
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
			createRuleset := data.CreateRuleset{
				Name:         ruleset.Name,
				Target:       ruleset.Target,
				Enforcement:  ruleset.Enforcement,
				BypassActors: ruleset.BypassActors,
				Conditions:   ruleset.Conditions,
				Rules:        make([]data.CreateRules, len(ruleset.Rules)),
			}
			zap.S().Debugf("Removing omitempty from fields if needed from ruleset: %s", ruleset.Name)
			for i, rule := range ruleset.Rules {
				if rule.Parameters == nil {
					createRuleset.Rules[i].Type = rule.Type
					continue
				} else {
					v := reflect.ValueOf(rule.Parameters).Elem()
					t := v.Type()

					newFields := make([]reflect.StructField, v.NumField())
					for j := 0; j < v.NumField(); j++ {
						fieldType := t.Field(j)
						fieldName := fieldType.Name

						if fields, ok := data.NonOmitEmptyFields[rule.Type]; ok {
							if utils.Contains(fields, fieldName) {
								fieldType = utils.UpdateTag(fieldType, "json", fieldName)
							}
						}
						newFields[j] = fieldType
					}
					newStructType := reflect.StructOf(newFields)
					newStruct := reflect.New(newStructType).Elem()
					for j := 0; j < v.NumField(); j++ {
						fieldValue := v.Field(j)
						if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.String {
							newStruct.Field(j).Set(reflect.MakeSlice(fieldValue.Type(), fieldValue.Len(), fieldValue.Cap()))
							for k := 0; k < fieldValue.Len(); k++ {
								newStruct.Field(j).Index(k).Set(fieldValue.Index(k))
							}
						} else {
							newStruct.Field(j).Set(fieldValue)
						}
					}
					createRuleset.Rules[i].Type = rule.Type
					createRuleset.Rules[i].Parameters = newStruct.Interface()
				}
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
				fmt.Printf("Successfully create repository ruleset %s for %s", ruleset.Name, owner)
			} else if ruleset.SourceType == "Repository" {
				zap.S().Debugf("Creating rulesets under %s", ruleset.Source)
				err = g.CreateRepoLevelRuleset(ruleset.Source, reader)
				if err != nil {
					zap.S().Errorf("Error arose creating ruleset %s for %s", ruleset.Name, ruleset.Source)
					continue
				}
				fmt.Printf("Successfully create repository ruleset %s for %s", ruleset.Name, ruleset.Source)
			}
		}
	} else {
		zap.S().Errorf("Error arose identifying rulesets and file data")
	}
	fmt.Printf("Successfully completed created repository rulesets from %s in org %s", cmdFlags.fileName, owner)
	return nil
}
