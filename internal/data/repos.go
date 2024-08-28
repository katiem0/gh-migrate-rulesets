package data

type AppInfo struct {
	AppID   int    `json:"id"`
	AppSlug string `json:"slug"`
}

type AppInstallation struct {
	InstallationID int    `json:"id"`
	AppID          int    `json:"app_id"`
	AppSlug        string `json:"app_slug"`
}

type AppIntegrations struct {
	TotalCount    int               `json:"total_count"`
	Installations []AppInstallation `json:"installations"`
}

type CustomRepoRoles struct {
	TotalCount  int          `json:"total_count"`
	CustomRoles []CustomRole `json:"custom_roles"`
}

type CustomRole struct {
	Name     string `json:"name"`
	ID       int    `json:"id"`
	BaseRole string `json:"base_role"`
}

type OrgIdQuery struct {
	Organization struct {
		DatabaseID int `json:"databaseId"`
	} `graphql:"organization(login: $owner)"`
}

type OrgID struct {
	DatabaseID int `json:"databaseId"`
}

type RepoInfo struct {
	DatabaseId int    `json:"databaseId"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type ReposQuery struct {
	Organization struct {
		Repositories struct {
			TotalCount int
			Nodes      []RepoInfo
			PageInfo   struct {
				EndCursor   string
				HasNextPage bool
			}
		} `graphql:"repositories(first: 100, after: $endCursor)"`
	} `graphql:"organization(login: $owner)"`
}

type RepoSingleQuery struct {
	Repository RepoInfo `graphql:"repository(owner: $owner, name: $name)"`
}

type TeamInfo struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}
