package goodsservice

import (
	"context"
	"fmt"
	"time"

	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/rs/zerolog/log"
)

type goodsStore interface {
	CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error)
	GetGood(ctx context.Context, id int, project_id int) (entity.Good, error)
	UpdateGood(ctx context.Context, id int, project_id int, goodUpdate entity.GoodUpdate) (entity.Good, error)
	DeleteGood(ctx context.Context, id int, project_id int) (entity.GoodDeleteResponse, error)
	GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, error)
	Reprioritize(ctx context.Context, id int, project_id int, new_priority entity.PriorityRequest) (entity.PriorityResponse, error)
}

type NATSPublisher interface {
	Publish(subject string, data interface{}) error
}

type Service struct {
	goodsStore goodsStore
	natsClient NATSPublisher
}

func New(goodsStore goodsStore, natsClient NATSPublisher) *Service {
	return &Service{
		goodsStore: goodsStore,
		natsClient: natsClient,
	}
}

func (s *Service) CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error) {
	err := req.Validate()
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to validate good: %w", err)
	}

	if projectID <= 0 {
		return entity.Good{}, fmt.Errorf("project ID cannot be negative or zero")
	}

	createdGood, err := s.goodsStore.CreateGood(ctx, projectID, req)
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to create good: %w", err)
	}

	logMsg := entity.GoodLog{
		Operation:   "create",
		GoodID:      createdGood.ID,
		ProjectID:   createdGood.ProjectID,
		Name:        createdGood.Name,
		Description: createdGood.Description,
		Priority:    createdGood.Priority,
		Removed:     createdGood.Removed,
		EventTime:   time.Now(),
	}

	if err := s.natsClient.Publish("goods.logs", logMsg); err != nil {
		log.Debug().Err(err).Msg("failed to publish lo to NATS")
	}

	return createdGood, nil
}

func (s *Service) GetGood(ctx context.Context, id int, project_id int) (entity.Good, error)
func (s *Service) UpdateGood(ctx context.Context, id int, project_id int, goodUpdate entity.GoodUpdate) (entity.Good, error)
func (s *Service) DeleteGood(ctx context.Context, id int, project_id int) (entity.GoodDeleteResponse, error)
func (s *Service) GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, error)
func (s *Service) Reprioritize(ctx context.Context, id int, project_id int, new_priority entity.PriorityRequest) (entity.PriorityResponse, error)
