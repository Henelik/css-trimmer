package diff

import (
	"path"
	"regexp"
	"slices"

	"github.com/Henelik/css-trimmer/internal/config"
)

type DiffResult struct {
	Used        []string
	Unused      []string
	Whitelisted []string
	Blacklisted []string
	ToRemove    []string
}

// Compute calculates which classes should be removed.
func Compute(inventory, usedClasses []string, cfg *config.Config) *DiffResult {
	result := &DiffResult{
		Used:        make([]string, 0, len(inventory)),
		Unused:      make([]string, 0, len(inventory)),
		Whitelisted: make([]string, 0, len(inventory)),
		Blacklisted: make([]string, 0, len(inventory)),
		ToRemove:    make([]string, 0, len(inventory)),
	}

	// Build sets
	usedSet := buildUsedSet(inventory, usedClasses, cfg.DynamicClassPatterns)
	whitelistSet := buildWhitelistSet(inventory, cfg.Whitelist)
	blacklistSet := buildBlacklistSet(inventory, cfg.Blacklist)

	for _, className := range inventory {
		_, isUsed := usedSet[className]

		if isUsed {
			result.Used = append(result.Used, className)
		} else {
			result.Unused = append(result.Unused, className)
		}

		_, isWhitelisted := whitelistSet[className]
		_, isBlacklisted := blacklistSet[className]

		if isWhitelisted {
			result.Whitelisted = append(result.Whitelisted, className)
		}

		if isBlacklisted {
			result.Blacklisted = append(result.Blacklisted, className)
		}

		if isBlacklisted || (!isWhitelisted && !isUsed) {
			result.ToRemove = append(result.ToRemove, className)
		}
	}

	slices.Sort(result.ToRemove)
	slices.Sort(result.Used)
	slices.Sort(result.Unused)
	slices.Sort(result.Whitelisted)
	slices.Sort(result.Blacklisted)

	return result
}

// buildUsedSet creates a set of classes that appear in source files or match dynamic patterns.
func buildUsedSet(inventory, usedClasses, classPatterns []string) map[string]struct{} {
	usedSet := make(map[string]struct{})

	// Add explicitly found classes
	for _, className := range usedClasses {
		usedSet[className] = struct{}{}
	}

	// Add classes matching dynamic patterns
	for _, className := range inventory {
		if matchesDynamicPattern(className, classPatterns) {
			usedSet[className] = struct{}{}
		}
	}

	return usedSet
}

// matchesDynamicPattern checks if a class matches any dynamic pattern regex.
func matchesDynamicPattern(className string, classPatterns []string) bool {
	for _, pattern := range classPatterns {
		if matched, _ := regexp.MatchString(pattern, className); matched {
			return true
		}
	}

	return false
}

// buildWhitelistSet creates a set of whitelisted classes using glob patterns.
func buildWhitelistSet(inventory, whitelist []string) map[string]struct{} {
	whitelistSet := make(map[string]struct{})

	for _, className := range inventory {
		for _, pattern := range whitelist {
			if globMatch(pattern, className) {
				whitelistSet[className] = struct{}{}
				break
			}
		}
	}

	return whitelistSet
}

// buildBlacklistSet creates a set of blacklisted classes using glob patterns.
func buildBlacklistSet(inventory, blacklist []string) map[string]struct{} {
	blacklistSet := make(map[string]struct{})

	for _, className := range inventory {
		for _, pattern := range blacklist {
			if globMatch(pattern, className) {
				blacklistSet[className] = struct{}{}
				break
			}
		}
	}

	return blacklistSet
}

// globMatch uses path.Match semantics for glob patterns.
func globMatch(pattern, className string) bool {
	matched, _ := path.Match(pattern, className)
	return matched
}
