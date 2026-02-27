package git

import "github.com/mikanfactory/yakumo/internal/model"

// GetBranchDiffStat runs `git diff <base>...HEAD --numstat` and returns
// aggregated line insertion/deletion counts for the branch.
func GetBranchDiffStat(runner CommandRunner, worktreePath, baseRef string) (model.StatusInfo, error) {
	entries, err := GetDiffNumstat(runner, worktreePath, baseRef)
	if err != nil {
		return model.StatusInfo{}, err
	}

	var info model.StatusInfo
	for _, e := range entries {
		info.Insertions += e.Additions
		info.Deletions += e.Deletions
	}
	return info, nil
}
