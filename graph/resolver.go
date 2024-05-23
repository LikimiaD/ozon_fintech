package graph

import (
	"github.com/likimiad/ozon_fintech/internal/database/models"
	"sync"

	"github.com/likimiad/ozon_fintech/internal/database"
	"log/slog"
)

// Resolver struct includes PostService and a map for subscription channels
type Resolver struct {
	PostService   *database.PostService
	mu            sync.Mutex
	subscriptions map[uint]chan *models.Comment
}

// NewResolver initializes a new resolver with the provided PostService
func NewResolver(postService *database.PostService) *Resolver {
	slog.Info("Initializing new resolver")
	return &Resolver{
		PostService:   postService,
		subscriptions: make(map[uint]chan *models.Comment),
	}
}
