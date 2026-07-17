package data

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
	"3": "Unknown",
	"4": "Write",
	"5": "Admin",
}
