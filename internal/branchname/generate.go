package branchname

import (
	"regexp"
	"strings"

	"github.com/mikanfactory/yakumo/internal/git"
)

const maxBranchNameLength = 30

var validBranchChar = regexp.MustCompile(`[^a-z0-9-]`)
var multiHyphen = regexp.MustCompile(`-{2,}`)

// SanitizeBranchName ensures the name is kebab-case, lowercase, and within the max length.
func SanitizeBranchName(name string) string {
	result := git.Slugify(name)

	result = validBranchChar.ReplaceAllString(result, "")
	result = multiHyphen.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	if len(result) > maxBranchNameLength {
		result = result[:maxBranchNameLength]
		result = strings.TrimRight(result, "-")
	}

	return result
}
