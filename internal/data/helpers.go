package data

var HeaderMap = map[string]string{
	"RulesCreation":                 "creation",
	"RulesUpdate":                   "update",
	"RulesDeletion":                 "deletion",
	"RulesRequiredLinearHistory":    "required_linear_history",
	"RulesMergeQueue":               "merge_queue",
	"RulesRequiredDeployments":      "required_deployments",
	"RulesRequiredSignatures":       "required_signatures",
	"RulesPullRequest":              "pull_request",
	"RulesRequiredStatusChecks":     "required_status_checks",
	"RulesNonFastForward":           "non_fast_forward",
	"RulesCommitMessagePattern":     "commit_message_pattern",
	"RulesCommitAuthorEmailPattern": "commit_author_email_pattern",
	"RulesCommitterEmailPattern":    "committer_email_pattern",
	"RulesBranchNamePattern":        "branch_name_pattern",
	"RulesTagNamePattern":           "tag_name_pattern",
	"RulesFilePathRestriction":      "file_path_restriction",
	"RulesFilePathLength":           "max_file_path_length",
	"RulesFileExtensionRestriction": "file_extension_restriction",
	"RulesMaxFileSize":              "max_file_size",
	"RulesWorkflows":                "workflows",
	"RulesCodeScanning":             "code_scanning",
}

var NonOmitEmptyFields = map[string][]string{
	"merge_queue":                 {"CheckResponseTimeoutMinutes", "GroupingStrategy", "MaxEntriesToBuild", "MaxEntriesToMerge", "MergeMethod", "MinEntriesToMerge", "MinEntriesToMergeWaitMinutes"},
	"required_deployments":        {"RequiredDeploymentEnvironments"},
	"pull_request":                {"DismissStaleReviewsOnPush", "RequireCodeOwnerReview", "RequireLastPushApproval", "RequiredApprovingReviewCount", "RequiredReviewThreadResolution"},
	"required_status_checks":      {"RequiredStatusChecks", "StrictRequiredStatusChecksPolicy"},
	"commit_message_pattern":      {"Name", "Negate", "Operator", "Pattern"},
	"commit_author_email_pattern": {"Name", "Negate", "Operator", "Pattern"},
	"committer_email_pattern":     {"Name", "Negate", "Operator", "Pattern"},
	"branch_name_pattern":         {"Name", "Negate", "Operator", "Pattern"},
	"tag_name_pattern":            {"Name", "Negate", "Operator", "Pattern"},
	"file_path_restriction":       {"RestrictedFilePaths"},
	"code_scanning":               {"CodeScanningTools"},
	"max_file_path_length":        {"MaxFilePathLength"},
	"file_extension_restriction":  {"RestrictedFileExtensions"},
	"max_file_size":               {"MaxFileSize"},
	"workflows":                   {"Workflows"},
	"update":                      {"update_allows_fetch_and_merge"},
}

type ProcessedConditions struct {
	IncludeNames    string
	ExcludeNames    string
	BoolNames       string
	PropertyInclude []string
	PropertyExclude []string
	IncludeRefNames string
	ExcludeRefNames string
}

var RolesMap = map[string]string{
	"0": "AllDeployKeys",
	"1": "OrgAdmin",
	"2": "Maintainer",
	"4": "Write",
	"5": "Admin",
}
