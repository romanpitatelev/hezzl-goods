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

type redisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type Service struct {
	goodsStore  goodsStore
	natsClient  NATSPublisher
	redisClient redisClient
}

func New(goodsStore goodsStore, natsClient NATSPublisher, redredisClient redisClient) *Service {
	return &Service{
		goodsStore:  goodsStore,
		natsClient:  natsClient,
		redisClient: redredisClient,
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
		log.Warn().Err(err).Msg("failed to publish new good to NATS")
	}

	return createdGood, nil
}

func (s *Service) GetGood(ctx context.Context, id int, projectID int) (entity.Good, error)

func (s *Service) UpdateGood(ctx context.Context, id int, projectID int, goodUpdate entity.GoodUpdate) (entity.Good, error) {
	if id <= 0 || projectID <= 0 {
		return entity.Good{}, entity.ErrInvalidIDOrProjectID
	}

	if goodUpdate.Name == "" {
		return entity.Good{}, entity.ErrEmptyName
	}

	updatedGood, err := s.goodsStore.UpdateGood(ctx, id, projectID, goodUpdate)
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to update good: %w", err)
	}

	cacheKeys := []string{
		fmt.Sprintf("good:%d:%d", id, projectID),
		"goods:list:*",
	}

	if err := s.redisClient.Del(ctx, cacheKeys...); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate redis cache")
	}

	logMsg := entity.GoodLog{
		Operation:   "update",
		GoodID:      updatedGood.ID,
		ProjectID:   updatedGood.ProjectID,
		Name:        updatedGood.Name,
		Description: updatedGood.Description,
		Priority:    updatedGood.Priority,
		Removed:     updatedGood.Removed,
		EventTime:   time.Now(),
	}

	if err := s.natsClient.Publish("goods.logs", logMsg); err != nil {
		log.Warn().Err(err).Msg("failed to publish updated good to NATS")
	}

	return updatedGood, nil
}

func (s *Service) DeleteGood(ctx context.Context, id int, project_id int) (entity.GoodDeleteResponse, error)
func (s *Service) GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, error)
func (s *Service) Reprioritize(ctx context.Context, id int, project_id int, new_priority entity.PriorityRequest) (entity.PriorityResponse, error)
