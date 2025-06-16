package goodsservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/rs/zerolog/log"
)

type goodsStore interface {
	CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error)
	GetGood(ctx context.Context, id int, projectID int) (entity.Good, error)
	UpdateGood(ctx context.Context, id int, projectID int, goodUpdate entity.GoodUpdate) (entity.Good, error)
	DeleteGood(ctx context.Context, id int, projectID int) (entity.GoodDeleteResponse, error)
	GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, entity.Meta, error)
	Reprioritize(ctx context.Context, id int, projectID int, newPriority entity.PriorityRequest) ([]entity.Priority, error)
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
		return entity.Good{}, entity.ErrInvalidIDOrProjectID
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

func (s *Service) GetGood(ctx context.Context, id int, projectID int) (entity.Good, error) {
	if id <= 0 || projectID <= 0 {
		return entity.Good{}, entity.ErrInvalidIDOrProjectID
	}

	cacheKey := fmt.Sprintf("good:%d:%d", id, projectID)
	cached, err := s.redisClient.Get(ctx, cacheKey)

	if err == nil && cached != "" {
		var good entity.Good
		if err := json.Unmarshal([]byte(cached), &good); err == nil {
			return good, nil
		}
	}

	good, err := s.goodsStore.GetGood(ctx, id, projectID)
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to get good: %w", err)
	}

	jsonData, err := json.Marshal(good)
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to marshal good for caching: %w", err)
	}

	if err := s.redisClient.Set(ctx, cacheKey, jsonData, time.Minute); err != nil {
		log.Warn().Err(err).Msg("failed to cache good")
	}

	logMsg := entity.GoodLog{
		Operation:   "get",
		GoodID:      good.ID,
		ProjectID:   good.ProjectID,
		Name:        good.Name,
		Description: good.Description,
		Priority:    good.Priority,
		Removed:     good.Removed,
		EventTime:   time.Now(),
	}

	if err := s.natsClient.Publish("goods.logs", logMsg); err != nil {
		log.Warn().Err(err).Msg("failed to publish get good to NATS")
	}

	return good, nil
}

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

func (s *Service) DeleteGood(ctx context.Context, id int, projectID int) (entity.GoodDeleteResponse, error) {
	if id <= 0 || projectID <= 0 {
		return entity.GoodDeleteResponse{}, entity.ErrInvalidIDOrProjectID
	}

	deletedGood, err := s.goodsStore.DeleteGood(ctx, id, projectID)
	if err != nil {
		return entity.GoodDeleteResponse{}, fmt.Errorf("failed to delete good: %w", err)
	}

	cacheKeys := []string{
		fmt.Sprintf("good:%d:%d", id, projectID),
		"goods:list:*",
	}
	if err := s.redisClient.Del(ctx, cacheKeys...); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate redis cache while deleting")
	}

	logMsg := entity.GoodLog{
		Operation:   "delete",
		GoodID:      deletedGood.ID,
		ProjectID:   deletedGood.CampaignID,
		Name:        "",
		Description: "",
		Priority:    0,
		Removed:     true,
		EventTime:   time.Now(),
	}

	if err := s.natsClient.Publish("goods.logs", logMsg); err != nil {
		log.Warn().Err(err).Msg("failed to publish deleted good to NATS")
	}

	return deletedGood, nil
}

func (s *Service) GetGoods(ctx context.Context, request entity.ListRequest) (entity.GoodsListResponse, error) {
	request.Validate()

	cacheKey := fmt.Sprintf("goods:list:%d:%d", request.Limit, request.Offset)
	cached, err := s.redisClient.Get(ctx, cacheKey)

	if err != nil && cached != "" {
		var response entity.GoodsListResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return response, nil
		}

		log.Warn().Err(err).Msg("error unmarshalling cached goods")
	}

	goods, meta, err := s.goodsStore.GetGoods(ctx, request)
	if err != nil {
		return entity.GoodsListResponse{}, fmt.Errorf("failed to get goods: %w", err)
	}

	response := entity.GoodsListResponse{
		Meta:  meta,
		Goods: goods,
	}

	jsonData, err := json.Marshal(response)
	if err == nil {
		if err := s.redisClient.Set(ctx, cacheKey, jsonData, time.Minute); err != nil {
			log.Warn().Err(err).Msg("failed to cache goods list")
		}
	}

	return response, nil
}

func (s *Service) Reprioritize(ctx context.Context, id int, projectID int, req entity.PriorityRequest) (entity.PriorityResponse, error) {
	if id <= 0 || projectID <= 0 {
		return entity.PriorityResponse{}, entity.ErrInvalidIDOrProjectID
	}

	if req.NewPriority <= 0 {
		return entity.PriorityResponse{}, entity.ErrNegativePriority
	}

	updatedPriorities, err := s.goodsStore.Reprioritize(ctx, id, projectID, req)
	if err != nil {
		return entity.PriorityResponse{}, fmt.Errorf("failed to reprioritize: %w", err)
	}

	cacheKeys := make([]string, 0, len(updatedPriorities)+1)
	for _, p := range updatedPriorities {
		cacheKeys = append(cacheKeys, fmt.Sprintf("good:%d:%d", p.ID, projectID))
	}

	cacheKeys = append(cacheKeys, "goods:list:*")

	if err := s.redisClient.Del(ctx, cacheKeys...); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate redis cache")
	}

	for _, p := range updatedPriorities {
		logMsg := entity.GoodLog{
			Operation: "reprioritize",
			GoodID:    p.ID,
			ProjectID: projectID,
			Priority:  p.Priority,
			EventTime: time.Now(),
		}

		if err := s.natsClient.Publish("goods.logs", logMsg); err != nil {
			log.Warn().Err(err).Msgf("failed to publish log for good %d", p.ID)
		}
	}

	return entity.PriorityResponse{
		Priorities: updatedPriorities,
	}, nil
}
