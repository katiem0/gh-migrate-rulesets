package data

// RuleSpec is the single source of truth for a ruleset rule type. It ties the
// CSV column header to the GitHub API rule type, the ruleset targets the rule is
// valid for, the parameter fields that should be read/written, and the fields
// that must not be omitted (omitempty) when a ruleset is imported.
type RuleSpec struct {
	Type         string
	CSVHeader    string
	Targets      []string
	Fields       map[string]map[string]struct{}
	NonOmitEmpty []string
}

// Ruleset targets.
const (
	TargetBranch     = "branch"
	TargetTag        = "tag"
	TargetPush       = "push"
	TargetRepository = "repository"
)

// RuleRegistry is the ordered catalog of all supported rule types. The order
// defines the CSV rule-column order and must remain stable so previously
// exported CSVs continue to import correctly.
var RuleRegistry = []RuleSpec{
	{
		Type:      "creation",
		CSVHeader: "RulesCreation",
		Targets:   []string{TargetBranch, TargetTag},
	},
	{
		Type:      "update",
		CSVHeader: "RulesUpdate",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"UpdateAllowsFetchAndMerge": {},
		},
		NonOmitEmpty: []string{"UpdateAllowsFetchAndMerge"},
	},
	{
		Type:      "deletion",
		CSVHeader: "RulesDeletion",
		Targets:   []string{TargetBranch, TargetTag},
	},
	{
		Type:      "required_linear_history",
		CSVHeader: "RulesRequiredLinearHistory",
		Targets:   []string{TargetBranch},
	},
	{
		Type:      "merge_queue",
		CSVHeader: "RulesMergeQueue",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"CheckResponseTimeoutMinutes":  {},
			"GroupingStrategy":             {},
			"MaxEntriesToBuild":            {},
			"MaxEntriesToMerge":            {},
			"MergeMethod":                  {},
			"MinEntriesToMerge":            {},
			"MinEntriesToMergeWaitMinutes": {},
		},
		NonOmitEmpty: []string{"CheckResponseTimeoutMinutes", "GroupingStrategy", "MaxEntriesToBuild", "MaxEntriesToMerge", "MergeMethod", "MinEntriesToMerge", "MinEntriesToMergeWaitMinutes"},
	},
	{
		Type:      "required_deployments",
		CSVHeader: "RulesRequiredDeployments",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"RequiredDeploymentEnvironments": {},
		},
		NonOmitEmpty: []string{"RequiredDeploymentEnvironments"},
	},
	{
		Type:      "required_signatures",
		CSVHeader: "RulesRequiredSignatures",
		Targets:   []string{TargetBranch, TargetTag},
	},
	{
		Type:      "pull_request",
		CSVHeader: "RulesPullRequest",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"AllowedMergeMethods":       {},
			"DismissStaleReviewsOnPush": {},
			"DismissalRestriction": {
				"AllowedActors": {},
				"Enabled":       {},
			},
			"RequireCodeOwnerReview":         {},
			"RequireLastPushApproval":        {},
			"RequiredApprovingReviewCount":   {},
			"RequiredReviewThreadResolution": {},
			"RequiredReviewers": {
				"FilePatterns":     {},
				"MinimumApprovals": {},
				"Reviewer":         {},
			},
		},
		NonOmitEmpty: []string{"DismissStaleReviewsOnPush", "RequireCodeOwnerReview", "RequireLastPushApproval", "RequiredApprovingReviewCount", "RequiredReviewThreadResolution"},
	},
	{
		Type:      "required_status_checks",
		CSVHeader: "RulesRequiredStatusChecks",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"DoNotEnforceOnCreate": {},
			"RequiredStatusChecks": {
				"Context":       {},
				"IntegrationID": {},
			},
			"StrictRequiredStatusChecksPolicy": {},
		},
		NonOmitEmpty: []string{"RequiredStatusChecks", "StrictRequiredStatusChecksPolicy"},
	},
	{
		Type:      "non_fast_forward",
		CSVHeader: "RulesNonFastForward",
		Targets:   []string{TargetBranch, TargetTag},
	},
	{
		Type:      "commit_message_pattern",
		CSVHeader: "RulesCommitMessagePattern",
		Targets:   []string{TargetBranch, TargetTag},
		Fields: map[string]map[string]struct{}{
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		NonOmitEmpty: []string{"Name", "Negate", "Operator", "Pattern"},
	},
	{
		Type:      "commit_author_email_pattern",
		CSVHeader: "RulesCommitAuthorEmailPattern",
		Targets:   []string{TargetBranch, TargetTag},
		Fields: map[string]map[string]struct{}{
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		NonOmitEmpty: []string{"Name", "Negate", "Operator", "Pattern"},
	},
	{
		Type:      "committer_email_pattern",
		CSVHeader: "RulesCommitterEmailPattern",
		Targets:   []string{TargetBranch, TargetTag},
		Fields: map[string]map[string]struct{}{
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		NonOmitEmpty: []string{"Name", "Negate", "Operator", "Pattern"},
	},
	{
		Type:      "branch_name_pattern",
		CSVHeader: "RulesBranchNamePattern",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		NonOmitEmpty: []string{"Name", "Negate", "Operator", "Pattern"},
	},
	{
		Type:      "tag_name_pattern",
		CSVHeader: "RulesTagNamePattern",
		Targets:   []string{TargetTag},
		Fields: map[string]map[string]struct{}{
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		NonOmitEmpty: []string{"Name", "Negate", "Operator", "Pattern"},
	},
	{
		Type:      "file_path_restriction",
		CSVHeader: "RulesFilePathRestriction",
		Targets:   []string{TargetPush},
		Fields: map[string]map[string]struct{}{
			"RestrictedFilePaths": {},
		},
		NonOmitEmpty: []string{"RestrictedFilePaths"},
	},
	{
		Type:      "max_file_path_length",
		CSVHeader: "RulesFilePathLength",
		Targets:   []string{TargetPush},
		Fields: map[string]map[string]struct{}{
			"MaxFilePathLength": {},
		},
		NonOmitEmpty: []string{"MaxFilePathLength"},
	},
	{
		Type:      "file_extension_restriction",
		CSVHeader: "RulesFileExtensionRestriction",
		Targets:   []string{TargetPush},
		Fields: map[string]map[string]struct{}{
			"RestrictedFileExtensions": {},
		},
		NonOmitEmpty: []string{"RestrictedFileExtensions"},
	},
	{
		Type:      "max_file_size",
		CSVHeader: "RulesMaxFileSize",
		Targets:   []string{TargetPush},
		Fields: map[string]map[string]struct{}{
			"MaxFileSize": {},
		},
		NonOmitEmpty: []string{"MaxFileSize"},
	},
	{
		Type:      "workflows",
		CSVHeader: "RulesWorkflows",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"DoNotEnforceOnCreate": {},
			"Workflows": {
				"Path":         {},
				"Ref":          {},
				"RepositoryID": {},
				"SHA":          {},
			},
		},
		NonOmitEmpty: []string{"Workflows"},
	},
	{
		Type:      "code_scanning",
		CSVHeader: "RulesCodeScanning",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"CodeScanningTools": {
				"Tool":                    {},
				"SecurityAlertsThreshold": {},
				"AlertsThreshold":         {},
			},
		},
		NonOmitEmpty: []string{"CodeScanningTools"},
	},
	{
		Type:      "code_quality",
		CSVHeader: "RulesCodeQuality",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"Severity": {},
		},
	},
	{
		Type:      "copilot_code_review",
		CSVHeader: "RulesCopilotCodeReview",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"ReviewDraftPullRequests": {},
			"ReviewOnPush":            {},
		},
	},
	{
		Type:      "license_compliance_scanning",
		CSVHeader: "RulesLicenseComplianceScanning",
		Targets:   []string{TargetBranch},
	},
	{
		Type:      "code_coverage",
		CSVHeader: "RulesCodeCoverage",
		Targets:   []string{TargetBranch},
		Fields: map[string]map[string]struct{}{
			"MinimumCoverage": {},
			"MaxCoverageDrop": {},
		},
		NonOmitEmpty: []string{"MinimumCoverage", "MaxCoverageDrop"},
	},
}

// HeaderMap maps a CSV rule column header to its GitHub API rule type. Derived
// from RuleRegistry so it stays in sync with the single source of truth.
var HeaderMap = buildHeaderMap()

// NonOmitEmptyFields maps a rule type to the parameter fields whose omitempty
// JSON tag must be stripped on import. Derived from RuleRegistry.
var NonOmitEmptyFields = buildNonOmitEmptyFields()

func buildHeaderMap() map[string]string {
	m := make(map[string]string, len(RuleRegistry))
	for _, spec := range RuleRegistry {
		m[spec.CSVHeader] = spec.Type
	}
	return m
}

func buildNonOmitEmptyFields() map[string][]string {
	m := make(map[string][]string)
	for _, spec := range RuleRegistry {
		if len(spec.NonOmitEmpty) > 0 {
			m[spec.Type] = spec.NonOmitEmpty
		}
	}
	return m
}

// RuleHeaders returns the CSV rule column headers in registry order.
func RuleHeaders() []string {
	headers := make([]string, 0, len(RuleRegistry))
	for _, spec := range RuleRegistry {
		headers = append(headers, spec.CSVHeader)
	}
	return headers
}

// RuleSpecByType returns the RuleSpec for a rule type, or nil if unknown.
func RuleSpecByType(ruleType string) *RuleSpec {
	for i := range RuleRegistry {
		if RuleRegistry[i].Type == ruleType {
			return &RuleRegistry[i]
		}
	}
	return nil
}

// RuleSpecByHeader returns the RuleSpec for a CSV header, or nil if unknown.
func RuleSpecByHeader(header string) *RuleSpec {
	for i := range RuleRegistry {
		if RuleRegistry[i].CSVHeader == header {
			return &RuleRegistry[i]
		}
	}
	return nil
}

// ValidFields returns the valid parameter fields for a rule type, or nil if the
// rule type is unknown or has no configurable parameters.
func ValidFields(ruleType string) map[string]map[string]struct{} {
	spec := RuleSpecByType(ruleType)
	if spec == nil {
		return nil
	}
	return spec.Fields
}
