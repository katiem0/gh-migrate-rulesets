package data

type BypassActor struct {
	ActorID    *int   `json:"actor_id,omitempty"`
	ActorType  string `json:"actor_type"`
	BypassMode string `json:"bypass_mode"`
}

type Conditions struct {
	RefName            *RefPatterns      `json:"ref_name,omitempty"`
	RepositoryName     *NamePatterns     `json:"repository_name,omitempty"`
	RepositoryProperty *PropertyPatterns `json:"repository_property,omitempty"`
}

type CreateRuleset struct {
	Name         string        `json:"name"`
	Target       string        `json:"target"`
	Enforcement  string        `json:"enforcement"`
	BypassActors []BypassActor `json:"bypass_actors,omitempty"`
	Conditions   *Conditions   `json:"conditions,omitempty"`
	Rules        []CreateRules `json:"rules"`
}

type CreateRules struct {
	Type       string      `json:"type"`
	Parameters interface{} `json:"parameters,omitempty"`
}

type OrgRulesetsQuery struct {
	Organization struct {
		Rulesets struct {
			Nodes    []Rulesets
			PageInfo struct {
				EndCursor   string
				HasNextPage bool
			}
		} `graphql:"rulesets(first: 100, after: $endCursor, includeParents:false)"`
	} `graphql:"organization(login: $owner)"`
}

type Rulesets struct {
	ID         string `json:"id"`
	DatabaseID int    `json:"databaseId"`
	Name       string `json:"name"`
}

type RepoNameRule struct {
	RepoName string
	Rule     Rulesets
}

type RepoRuleset struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Target       string        `json:"target"`
	SourceType   string        `json:"source_type"`
	Source       string        `json:"source"`
	Enforcement  string        `json:"enforcement"`
	BypassActors []BypassActor `json:"bypass_actors"`
	Conditions   *Conditions   `json:"conditions"`
	Rules        []Rules       `json:"rules"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
}

type RepoRulesetsQuery struct {
	Repository struct {
		Rulesets struct {
			Nodes    []Rulesets
			PageInfo struct {
				EndCursor   string
				HasNextPage bool
			}
		} `graphql:"rulesets(first: 100, after: $endCursor, includeParents:false)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type Rules struct {
	Type       string      `json:"type"`
	Parameters *Parameters `json:"parameters,omitempty"`
}

type RefPatterns struct {
	Exclude []string `json:"exclude"`
	Include []string `json:"include"`
}

type NamePatterns struct {
	Exclude   []string `json:"exclude"`
	Include   []string `json:"include"`
	Protected bool     `json:"protected"`
}

type PropertyPatterns struct {
	Exclude []PropertyPattern `json:"exclude"`
	Include []PropertyPattern `json:"include"`
}

type PropertyPattern struct {
	Name           string   `json:"name"`
	Source         string   `json:"source"`
	PropertyValues []string `json:"property_values"`
}

type Parameters struct {
	RequiredApprovingReviewCount     int            `json:"required_approving_review_count,omitempty"`
	DismissStaleReviewsOnPush        bool           `json:"dismiss_stale_reviews_on_push,omitempty"`
	RequireCodeOwnerReview           bool           `json:"require_code_owner_review,omitempty"`
	RequireLastPushApproval          bool           `json:"require_last_push_approval,omitempty"`
	RequiredReviewThreadResolution   bool           `json:"required_review_thread_resolution,omitempty"`
	DoNotEnforceOnCreate             bool           `json:"do_not_enforce_on_create,omitempty"`
	Workflows                        []Workflows    `json:"workflows,omitempty"`
	UpdateAllowsFetchAndMerge        bool           `json:"update_allows_fetch_and_merge,omitempty"`
	CheckResponseTimeoutMinutes      int            `json:"check_response_timeout_minutes,omitempty"`
	GroupingStrategy                 string         `json:"grouping_strategy,omitempty"`
	MaxEntriesToBuild                int            `json:"max_entries_to_build,omitempty"`
	MaxEntriesToMerge                int            `json:"max_entries_to_merge,omitempty"`
	MergeMethod                      string         `json:"merge_method,omitempty"`
	MinEntriesToMerge                int            `json:"min_entries_to_merge,omitempty"`
	MinEntriesToMergeWaitMinutes     int            `json:"min_entries_to_merge_wait_minutes,omitempty"`
	RequiredDeploymentEnvironments   []string       `json:"required_deployment_environments,omitempty"`
	RequiredStatusChecks             []StatusChecks `json:"required_status_checks,omitempty"`
	StrictRequiredStatusChecksPolicy bool           `json:"strict_required_status_checks_policy,omitempty"`
	Name                             string         `json:"name,omitempty"`
	Negate                           bool           `json:"negate,omitempty"`
	Operator                         string         `json:"operator,omitempty"`
	Pattern                          string         `json:"pattern,omitempty"`
	RestrictedFilePaths              []string       `json:"restricted_file_paths,omitempty"`
	MaxFilePathLength                int            `json:"max_file_path_length,omitempty"`
	RestrictedFileExtensions         []string       `json:"restricted_file_extensions,omitempty"`
	MaxFileSize                      int            `json:"max_file_size,omitempty"`
	CodeScanningTools                []CodeScanning `json:"code_scanning_tools,omitempty"`
}

type StatusChecks struct {
	Context       string `json:"context,omitempty"`
	IntegrationID *int   `json:"integration_id,omitempty"`
}

type CodeScanning struct {
	Tool                    string `json:"tool,omitempty"`
	SecurityAlertsThreshold string `json:"security_alerts_threshold,omitempty"`
	AlertsThreshold         string `json:"alerts_threshold,omitempty"`
}

type Workflows struct {
	Path         string `json:"path,omitempty"`
	Ref          string `json:"ref,omitempty"`
	RepositoryID int    `json:"repository_id,omitempty"`
	SHA          string `json:"sha,omitempty"`
}

type ErrorRulesets struct {
	Source      string
	RulesetName string
	Error       string
}
