package diff

import (
	"path"
	"regexp"
	"sort"

	"github.com/Henelik/css-trimmer/internal/config"
	"github.com/Henelik/css-trimmer/internal/css"
)

// Engine computes the set of classes to remove based on defined, used, and config rules.
type Engine struct {
	defined   []string
	used      []string
	config    *config.Config
	inventory css.ClassInventory
}

// NewEngine creates a new diff engine.
func NewEngine(inventory css.ClassInventory, used []string, cfg *config.Config) *Engine {
	return &Engine{
		defined:   inventory.AllClasses(),
		used:      used,
		config:    cfg,
		inventory: inventory,
	}
}

// Compute calculates which classes should be removed.
func (e *Engine) Compute() *DiffResult {
	result := &DiffResult{
		Used:        []string{},
		Unused:      []string{},
		Whitelisted: []string{},
		Blacklisted: []string{},
		ToRemove:    []string{},
	}

	// Build sets
	usedSet := e.buildUsedSet()
	whitelistSet := e.buildWhitelistSet()
	blacklistSet := e.buildBlacklistSet()

	// Categorize all defined classes
	toRemoveSet := make(map[string]struct{})

	for _, className := range e.defined {
		_, isUsed := usedSet[className]
		_, isWhitelisted := whitelistSet[className]
		_, isBlacklisted := blacklistSet[className]

		if isUsed {
			result.Used = append(result.Used, className)
		} else {
			result.Unused = append(result.Unused, className)
		}

		if isWhitelisted {
			result.Whitelisted = append(result.Whitelisted, className)
		}

		if isBlacklisted {
			result.Blacklisted = append(result.Blacklisted, className)
		}

		// Determine if class should be removed
		// 1. Blacklist always wins
		if isBlacklisted {
			toRemoveSet[className] = struct{}{}
		} else if !isWhitelisted && !isUsed {
			// 2. Not whitelisted and not used = remove
			toRemoveSet[className] = struct{}{}
		}
	}

	// Convert set to sorted slice
	for className := range toRemoveSet {
		result.ToRemove = append(result.ToRemove, className)
	}
	sort.Strings(result.ToRemove)
	sort.Strings(result.Used)
	sort.Strings(result.Unused)
	sort.Strings(result.Whitelisted)
	sort.Strings(result.Blacklisted)

	return result
}

// buildUsedSet creates a set of classes that appear in source files or match dynamic patterns.
func (e *Engine) buildUsedSet() map[string]struct{} {
	usedSet := make(map[string]struct{})

	// Add explicitly found classes
	for _, className := range e.used {
		usedSet[className] = struct{}{}
	}

	// Add classes matching dynamic patterns
	for _, className := range e.defined {
		if e.matchesDynamicPattern(className) {
			usedSet[className] = struct{}{}
		}
	}

	return usedSet
}

// matchesDynamicPattern checks if a class matches any dynamic pattern regex.
func (e *Engine) matchesDynamicPattern(className string) bool {
	for _, pattern := range e.config.DynamicClassPatterns {
		if matched, _ := regexp.MatchString(pattern, className); matched {
			return true
		}
	}
	return false
}

// buildWhitelistSet creates a set of whitelisted classes using glob patterns.
func (e *Engine) buildWhitelistSet() map[string]struct{} {
	whitelistSet := make(map[string]struct{})

	for _, className := range e.defined {
		for _, pattern := range e.config.Whitelist {
			if e.globMatch(pattern, className) {
				whitelistSet[className] = struct{}{}
				break
			}
		}
	}

	return whitelistSet
}

// buildBlacklistSet creates a set of blacklisted classes using glob patterns.
func (e *Engine) buildBlacklistSet() map[string]struct{} {
	blacklistSet := make(map[string]struct{})

	for _, className := range e.defined {
		for _, pattern := range e.config.Blacklist {
			if e.globMatch(pattern, className) {
				blacklistSet[className] = struct{}{}
				break
			}
		}
	}

	return blacklistSet
}

// globMatch uses path.Match semantics for glob patterns.
func (e *Engine) globMatch(pattern, className string) bool {
	matched, _ := path.Match(pattern, className)
	return matched
}
