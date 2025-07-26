// Project: Cardano Valley
package cv

import "golang.org/x/exp/constraints"

var (
	LogoImage = "https://preeb.cloud/wp-content/uploads/2025/04/CardanoValleyLogo.png"
	IconImage = "https://preeb.cloud/wp-content/uploads/2025/04/CardanoValleyIcon.png"
)

func SliceDiff[T constraints.Ordered](listA []T, listB []T) []T {
	ma := make(map[T]struct{}, len(listA))
	var diffs []T

	for _, ka := range listA {
		ma[ka] = struct{}{}
	}

	for _, kb := range listB {
		if _, ok := ma[kb]; !ok {
			diffs = append(diffs, kb)
		}
	}

	return diffs
}

func SliceMatches[T constraints.Ordered](listA []T, listB []T) []T {
	ma := make(map[T]struct{}, len(listA))
	var sames []T

	for _, ka := range listA {
		ma[ka] = struct{}{}
	}

	for _, kb := range listB {
		if _, ok := ma[kb]; ok {
			sames = append(sames, kb)
		}
	}

	return sames
}

func TruncateMiddle(s string, max int) string {
	if len(s) <= max {
		return s
	}

	// Reserve characters for the beginning and end
	startLen := (max - 3) / 2
	endLen := max - 3 - startLen

	return s[:startLen] + "..." + s[len(s)-endLen:]
}
