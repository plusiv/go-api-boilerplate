package repositories

import "context"

// Transactioner runs a callback in a transaction, committing on success and rolling back on error.
// The transaction handle is valid only during the callback.
type Transactioner interface {
	Transaction(ctx context.Context, fn func(tx any) error) error
}

// WithTransaction binds a repository to a handle from its matching Transactioner.
// It returns an error for an invalid or incompatible handle.
type WithTransaction[T any] interface {
	WithTx(tx any) (T, error)
}

// TransactionalItemRepository supports atomic operations using a transaction handle.
type TransactionalItemRepository interface {
	ItemRepository
	WithTransaction[TransactionalItemRepository]
}
