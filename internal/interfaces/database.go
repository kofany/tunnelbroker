package interfaces

// Database definiuje interfejs dla operacji bazodanowych
type Database interface {
	IsPrefixInUse(prefix string) (bool, error)
}
