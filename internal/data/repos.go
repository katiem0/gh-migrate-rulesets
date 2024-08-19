package data

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
