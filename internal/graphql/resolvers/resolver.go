package resolvers

import catalogrepo "github.com/sidsharma96/SysEscape/internal/catalog/repo"

// Resolver holds dependencies for GraphQL resolvers.
type Resolver struct {
	CatalogRepo catalogrepo.RoomRepo
}
