package css

// ClassSet holds defined CSS class names
type ClassSet map[string]bool

// ClassInfo stores metadata about where a class is defined in the CSS
type ClassInfo struct {
	StartLine int
	EndLine   int
}

// ClassInventory maps class names to their definition locations
type ClassInventory map[string]ClassInfo
