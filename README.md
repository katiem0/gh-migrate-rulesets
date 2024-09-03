# gh-migrate-rulesets

A GitHub `gh` [CLI](https://cli.github.com/) extension to create a report containing repository rulesets for a single repository, list of repositories, and/or organization, as well as create repository rulesets from a file.

> [!NOTE]
> The authenticated user must be an organization owner and a GitHub Personal Access Token needs the `admin:read` scope at the organization level to use this CLI extension to it's fullest.

## Installation

1. Install the `gh` CLI - see the [installation](https://github.com/cli/cli#installation) instructions.

2. Install the extension:
   ```sh
   gh extension install katiem0/gh-migrate-rulesets
   ```

For more information: [`gh extension install`](https://cli.github.com/manual/gh_extension_install).

## Usage

The `gh-migrate-rulesets` extension supports `GitHub.com` and GitHub Enterprise Server, through the use of `--hostname` and the following commands:

```sh
$ gh migrate-rulesets -h
List and create repository/organization level rulesets for repositories in an organization.

Usage:
  migrate-rules [command]

Available Commands:
  create      Create repository rulesets
  list        Generate a report of rulesets for repositories and/or organization.

Flags:
  -h, --help   help for migrate-rules

Use "migrate-rules [command] --help" for more information about a command.
```

### List Repository Rulesets

The `gh migrate-rulesets list` command will create a csv report of repository rulesets for the specified `<organization>` and/or `[repo ..]` list, with the ability to specify the `--host-name` and `--token` associated to a Server instance. If only `<organization>` is provided, all repositories will be used.

To specify the type of ruleset to list, setting the `--ruleType` flag will either list:

- `all`: Organization level, and repository level rulesets
- `repoOnly`: Repository level rulesets for list of repos or all repos under `<organization>`
- `orgOnly`: Organization level rulesets only

```sh
$ gh migrate-rulesets list -h
Generate a report of rulesets for a list of repositories and/or organization.

Usage:
  migrate-rules list [flags] <organization> [repo ...]

Flags:
  -d, --debug                To debug logging
  -h, --help                 help for list
      --hostname string      GitHub Enterprise Server hostname (default "github.com")
  -o, --output-file string   Name of file to write CSV list to (default "ruleset-20240819094546.csv")
  -r, --ruleType string      List rulesets for a specific application or all: {all|repoOnly|orgOnly} (default "all")
  -t, --token string         GitHub Personal Access Token (default "gh auth token")
```

The output `csv` file contains the following information:

<details>
<summary><b>Click to Expand output <code>csv</code> file contents</b></summary>
<table>
<tr><th>Field Name</th><th>Description</th></tr>
<tr><td><code>RulesetLevel</code></td><td>Indicates whether the ruleset is at the organization or repository level.</td></tr>
<tr><td><code>RepositoryName</code></td><td>If repository level ruleset, the name of the repository where the data is extracted from. For Organization rulesets, this is `N/A`.</td></tr>
<tr><td><code>RuleID</code></td><td>Unique identifier for the rule.</td></tr>
<tr><td><code>RulesetName</code></td><td>Name of the ruleset.</td></tr>
<tr><td><code>Target</code></td><td>Indicates the type of ruleset, can be `branch`, `tag`, or `push`.</td></tr>
<tr><td><code>Enforcement</code></td><td>Enforcement level of the ruleset (e.g., `active`, `evaluate`, or `disabled`).</td></tr>
<tr><td><code>BypassActors</code></td><td>Actors who can bypass the ruleset, specified in the format `ID;Role;Name;Condition`.</td></tr>
<tr><td><code>ConditionsRefNameInclude</code></td><td>Array of `ref` names to include in the ruleset conditions.</td></tr>
<tr><td><code>ConditionsRefNameExclude</code></td><td>Array of `ref` names to exclude from the ruleset conditions.</td></tr>
<tr><td><code>ConditionsRepoNameInclude</code></td><td>Array of repository names to include in the ruleset conditions.</td></tr>
<tr><td><code>ConditionsRepoNameExclude</code></td><td>Array of repository names to exclude from the ruleset conditions.</td></tr>
<tr><td><code>ConditionsRepoNameProtected</code></td><td>Indicates whether renaming of target repositories is prevented.</td></tr>
<tr><td><code>ConditionRepoPropertyInclude</code></td><td>Array of repository properties values to include in the ruleset conditions.</td></tr>
<tr><td><code>ConditionRepoPropertyExclude</code></td><td>Array of repository properties values to exclude from the ruleset conditions.</td></tr>
<tr><td><code>RulesCreation</code></td><td>Only allow users with bypass permission to create matching refs.</td></tr>
<tr><td><code>RulesUpdate</code></td><td>Only allow users with bypass permissions to delete matching refs.</td></tr>
<tr><td><code>RulesDeletion</code></td><td>Prevent merge commits from being pushed to matching refs.</td></tr>
<tr><td><code>RulesRequiredLinearHistory</code></td><td>Prevent merge commits from being pushed to matching refs.</td></tr>
<tr><td><code>RulesMergeQueue</code></td><td>Merges must be performed via a merge queue. In the format `check_response_timeout_minutes|grouping_strategy|max_entries_to_build|max_entries_to_merge|merge_method|min_entries_to_merge|min_entries_to_merge_wait_minutes`</td></tr>
<tr><td><code>RulesRequiredDeployments</code></td><td>Choose which environments must be successfully deployed to before refs can be pushed into a ref that matches this rule. Includes `required_deployment_environments` array.</td></tr>
<tr><td><code>RulesRequiredSignatures</code></td><td>Commits pushed to matching refs must have verified signatures.</td></tr>
<tr><td><code>RulesPullRequest</code></td><td>Require all commits be made to a non-target branch and submitted via a pull request before they can be merged. In the format `dismiss_stale_reviews_on_push|require_code_owner_review|require_last_push_approval|required_approving_review_count|required_review_thread_resolution`</td></tr>
<tr><td><code>RulesRequiredStatusChecks</code></td><td>Choose which status checks must pass before the ref is updated. An array of required status check rules, in the format `do_not_enforce_on_create|required_status_checks:{context|integration}|strict_required_status_checks_policy`</td></tr>
<tr><td><code>RulesNonFastForward</code></td><td>Prevent users with push access from force pushing to refs.</td></tr>
<tr><td><code>RulesCommitMessagePattern</code></td><td>Indicates commit message patterns and matching. In the format `Name|Negate|Operator|Pattern`</td></tr>
<tr><td><code>RulesCommitAuthorEmailPattern</code></td><td>Indicates commit author email patterns and matching. In the format `Name|Negate|Operator|Pattern`</td></tr>
<tr><td><code>RulesCommitterEmailPattern</code></td><td>Indicates committer email patterns and matching. In the format `Name|Negate|Operator|Pattern`</td></tr>
<tr><td><code>RulesBranchNamePattern</code></td><td>Indicates branch name patterns and matching. In the format `Name|Negate|Operator|Pattern`</td></tr>
<tr><td><code>RulesTagNamePattern</code></td><td>Indicates tag name patterns and matching. In the format `Name|Negate|Operator|Pattern`</td></tr>
<tr><td><code>RulesFilePathRestriction</code></td><td>Prevent commits that include changes in specified file paths from being pushed to the commit graph.</td></tr>
<tr><td><code>RulesFilePathLength</code></td><td>Prevent commits that include file paths that exceed a specified character limit from being pushed to the commit graph.</td></tr>
<tr><td><code>RulesFileExtensionRestriction</code></td><td>Restrictions on file extensions for the ruleset.</td></tr>
<tr><td><code>RulesMaxFileSize</code></td><td>Maximum file size allowed to be pushed to the commit.</td></tr>
<tr><td><code>RulesWorkflows</code></td><td>Require all changes made to a targeted branch to pass the specified workflows before they can be merged. An array of workflow rules, in the format `do_not_enforce_on_create|workflows:{Path|ref|repository_id|sha}`</td></tr>
<tr><td><code>RulesCodeScanning</code></td><td>Choose which tools must provide code scanning results before the reference is updated. An array of code scanning rules in the format `{Tool|SecurityAlertsThreshold|AlertsThreshold}`</td></tr>
<tr><td><code>CreatedAt</code></td><td>Timestamp of when the ruleset was created.</td></tr>
<tr><td><code>UpdatedAt</code></td><td>Timestamp of when the ruleset was last updated.</td></tr>
</table>
</details>
   
### Create Repository Rulesets

Repository Rulesets can be created from a `csv` file using `--from-file` following the format outlined in [`gh-migrate-rulesets list`](#list-repository-rulesets), or specifying the `--source-org` and/or `--repos` to retrieve rulesets from.

> [!WARNING]
> If your rulesets include the following rules, ensure that the `csv` has been updated to point to the updated information under your organization:
>
> - Bypass Actors: Update Actor ID for Teams, Roles, and Integrations
> - Status Checks: Ensure Context name exists and update Integration ID
> - Code Scanning: Ensure Tool name exists
> - Workflows: Update Repository ID to point to the correct repo/workflow
> - Required Deployments: Ensure deployment names exist for the repository

```sh
$ gh migrate-rulesets create -h                                                   
Create repository rulesets at the repo and/or org level from a file or list.

Usage:
  migrate-rules create [flags] <organization>

Flags:
  -d, --debug                    To debug logging
  -f, --from-file string         Path and Name of CSV file to create rulesets from
  -h, --help                     help for create
      --hostname string          GitHub Enterprise Server hostname (default "github.com")
  -R, --repos strings            List of repositories names to recreate rulesets for separated by commas (i.e. repo1,repo2,repo3)
  -r, --ruleType string          List rulesets for a specific application or all: {all|repoOnly|orgOnly} (default "all")
      --source-hostname string   GitHub Enterprise Server hostname where rulesets are copied from (default "github.com")
  -s, --source-org string        Name of the Source Organization to copy rulesets from
  -p, --source-pat string        GitHub personal access token for Source Organization (default "gh auth token")
  -t, --token string             GitHub personal access token for organization to write to (default "gh auth token")
```

If specifying `--source-org` and/or `--repos`, the CLI extension will attempt to map the object based on name to the new ID under the target organization:

- Bypass Actors
  - Teams
  - Custom Repository Roles
  - Integrations
- Status Checks
  - Context
- Required Workflow
  - Repository


> [!NOTE]
> If a ruleset fails to be created, a ruleset's Source, Name, and Error will be written to a `csv` file in the current directory with the name format `<org>-ruleset-errors-<date>.csv`.