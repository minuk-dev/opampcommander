package port

import (
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

var (
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	ErrConnectionNotFound      = errors.New("connection not found")
)

type ConnectionUsecase interface {
	GetConnectionUsecase
	SetConnectionUsecase
	DeleteConnectionUsecase
	ListConnectionIDsUsecase
}

type GetConnectionUsecase interface {
	GetConnection(instanceUID uuid.UUID) (*model.Connection, error)
}

type SetConnectionUsecase interface {
	SetConnection(connection *model.Connection) error
}

type DeleteConnectionUsecase interface {
	DeleteConnection(instanceUID uuid.UUID) error
}

type ListConnectionIDsUsecase interface {
	ListConnectionIDs() []uuid.UUID
}
