package mongodb

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.CommandPersistencePort = (*CommandRepository)(nil)

const (
	commandCollectionName = "commands"
)

// CommandRepository is a struct that implements the CommandPersistencePort interface.
type CommandRepository struct {
	collection *mongo.Collection
}

// NewCommandRepository creates a new instance of CommandRepository.
func NewCommandRepository(
	mongoDatabase *mongo.Database,
) *CommandRepository {
	collection := mongoDatabase.Collection(commandCollectionName)

	return &CommandRepository{
		collection: collection,
	}
}

// ErrNotImplemented is an error that indicates that the method is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// GetCommand retrieves a command by its ID.
func (r *CommandRepository) GetCommand(context.Context, uuid.UUID) (*model.Command, error) {
	return nil, ErrNotImplemented
}

// GetCommandByInstanceUID retrieves a command by its instance UID.
func (r *CommandRepository) GetCommandByInstanceUID(context.Context, uuid.UUID) (*model.Command, error) {
	return nil, ErrNotImplemented
}

// SaveCommand saves the command to the persistence layer.
func (r *CommandRepository) SaveCommand(context.Context, *model.Command) error {
	return ErrNotImplemented
}
