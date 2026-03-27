package diff

// DiffResult contains the results of class analysis.
type DiffResult struct {
	Used        []string
	Unused      []string
	Whitelisted []string
	Blacklisted []string
	ToRemove    []string
}
