package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
)

func (s *IntegrationTestSuite) TestCreateGood() {
	s.Run("create one good successfully, but 0 logs in clickhouse", func() {
		good := entity.GoodCreateRequest{
			Name: "one",
		}

		path := goodsPath + "/create" + "?projectId=1"

		var createdGood entity.Good

		s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

		time.Sleep(100 * time.Millisecond)

		s.Require().Equal(1, createdGood.ProjectID)
		s.Require().Equal(good.Name, createdGood.Name)
		s.Require().Equal(1, createdGood.Priority)
		s.Require().False(createdGood.Removed)

		var logCount int
		err := s.clickhouseStore.DB().QueryRowContext(context.Background(),
			"SELECT count() FROM goods_logs WHERE id = $1 AND project_id = $2 AND operation = 'create'",
			createdGood.ID, createdGood.ProjectID).Scan(&logCount)
		s.Require().NoError(err)
		s.Require().Equal(0, logCount, "should have 0 logs")
	})

	s.Run("create 70 goods and load clickhouse with 2 batches of 30 successfully", func() {
		path := goodsPath + "/create" + "?projectId=1"

		for i := range 70 {
			good := entity.GoodCreateRequest{
				Name: fmt.Sprintf("one_%d", i),
			}

			var createdGood entity.Good

			s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

			time.Sleep(100 * time.Millisecond)
		}

		var logCount int
		err := s.clickhouseStore.DB().QueryRowContext(context.Background(),
			"SELECT count() FROM goods_logs WHERE project_id = $1 AND operation = 'create'",
			1).Scan(&logCount)
		s.Require().NoError(err)
		s.Require().Equal(60, logCount, "should have 60 logs")
	})

	s.Run("create good with empty name", func() {
		description := "nice good"
		good := entity.GoodCreateRequest{
			Description: &description,
		}

		path := goodsPath + "/create" + "?projectId=1"

		s.sendRequest(http.MethodPost, path, http.StatusBadRequest, &good, nil)
	})

	s.Run("create good with negative project id", func() {
		description := "nice good"
		good := entity.GoodCreateRequest{
			Name:        "negative project id",
			Description: &description,
		}

		path := goodsPath + "/create" + "?projectId=-20"

		s.sendRequest(http.MethodPost, path, http.StatusBadRequest, &good, nil)
	})
}

func (s *IntegrationTestSuite) TestGetGood() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	good := entity.GoodCreateRequest{
		Name: "test good",
	}

	path := goodsPath + "/create" + "?projectId=1"

	var createdGood entity.Good
	s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

	s.Run("get good successfully - good is cached in redis", func() {
		pathGet := goodsPath + fmt.Sprintf("/get?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)

		var goodFound entity.Good
		s.sendRequest(http.MethodGet, pathGet, http.StatusOK, nil, &goodFound)

		cacheKey := fmt.Sprintf("good:%d:%d", goodFound.ID, goodFound.ProjectID)
		cached, err := s.redisClient.Get(ctx, cacheKey)
		s.Require().NoError(err)

		var cachedGood entity.Good
		err = json.Unmarshal([]byte(cached), &cachedGood)
		s.Require().NoError(err)
		s.Require().Equal(goodFound.ID, cachedGood.ID)
		s.Require().Equal(goodFound.Priority, cachedGood.Priority)
		s.Require().Equal(goodFound.Name, cachedGood.Name)
	})

	s.Run("get good successfully - good is not in redis after one minue", func() {
		pathGet := goodsPath + fmt.Sprintf("/get?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)

		var goodFound entity.Good
		s.sendRequest(http.MethodGet, pathGet, http.StatusOK, nil, &goodFound)

		cacheKey := fmt.Sprintf("good:%d:%d", goodFound.ID, goodFound.ProjectID)

		_, err := s.redisClient.Get(ctx, cacheKey)
		s.Require().NoError(err, "cache should exist immediately afetr get")

		expirationChecked := false
		for start := time.Now(); time.Since(start) < 70*time.Second; {
			ttl, err := s.redisClient.TTL(ctx, cacheKey)
			if err != nil || ttl < 0 {
				expirationChecked = true
				break
			}
			time.Sleep(1 * time.Second)
		}
		s.Require().True(expirationChecked, "cache should expire within 60 seconds")

		_, err = s.redisClient.Get(ctx, cacheKey)
		s.Require().Error(err, "cache should be expired")
		s.Require().ErrorIs(err, redis.Nil)
	})

	s.Run("good not found", func() {
		pathGet := goodsPath + fmt.Sprintf("/get?id=%d&projectId=%d", 9999, createdGood.ProjectID)
		s.sendRequest(http.MethodGet, pathGet, http.StatusNotFound, nil, nil)
	})
}

func (s *IntegrationTestSuite) TestUpdateGood() {
	good := entity.GoodCreateRequest{
		Name: "test update good",
	}

	path := goodsPath + "/create" + "?projectId=1"

	var createdGood entity.Good
	s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

	s.Run("update good successfully", func() {
		description := "some update description"
		updateReq := entity.GoodUpdate{
			Name:        "updated-name",
			Description: &description,
		}

		path := goodsPath + fmt.Sprintf("/update?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)
		var updatedGood entity.Good
		s.sendRequest(http.MethodPatch, path, http.StatusOK, &updateReq, &updatedGood)

		s.Require().Equal(updateReq.Name, updatedGood.Name)
		s.Require().Equal(description, updatedGood.Description)
	})

	s.Run("update good and invalidate redis", func() {
		ctx := context.Background()

		pathGet := goodsPath + fmt.Sprintf("/get?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)

		var goodFound entity.Good
		s.sendRequest(http.MethodGet, pathGet, http.StatusOK, nil, &goodFound)

		cacheKey := fmt.Sprintf("good:%d:%d", goodFound.ID, goodFound.ProjectID)
		cached, err := s.redisClient.Get(ctx, cacheKey)
		s.Require().NoError(err)

		var cachedGood entity.Good
		err = json.Unmarshal([]byte(cached), &cachedGood)
		s.Require().NoError(err)
		s.Require().Equal(goodFound.ID, cachedGood.ID)
		s.Require().Equal(goodFound.Priority, cachedGood.Priority)
		s.Require().Equal(goodFound.Name, cachedGood.Name)

		description := "new update description"
		updateReq := entity.GoodUpdate{
			Name:        "new-updated-name",
			Description: &description,
		}

		path := goodsPath + fmt.Sprintf("/update?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)
		var updatedGood entity.Good
		s.sendRequest(http.MethodPatch, path, http.StatusOK, &updateReq, &updatedGood)

		s.Require().Equal(updateReq.Name, updatedGood.Name)
		s.Require().Equal(description, updatedGood.Description)

		_, err = s.redisClient.Get(ctx, cacheKey)
		s.Require().Error(err, "cache should be invalidated")
		s.Require().ErrorIs(err, redis.Nil)
	})

	s.Run("update good not found", func() {
		updateReq := entity.GoodUpdate{
			Name: "updated-name-not-found",
		}

		path := goodsPath + fmt.Sprintf("/update?id=%d&projectId=%d", 9999, createdGood.ProjectID)
		s.sendRequest(http.MethodPatch, path, http.StatusNotFound, &updateReq, nil)
	})
}

func (s *IntegrationTestSuite) TestDeleteGood() {
	good := entity.GoodCreateRequest{
		Name: "test delete good",
	}

	path := goodsPath + "/create" + "?projectId=1"

	var createdGood entity.Good
	s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

	s.Run("delete good successfully", func() {
		pathDelete := goodsPath + fmt.Sprintf("/remove?id=%d&projectId=%d", createdGood.ID, createdGood.ProjectID)

		var deleteResponse entity.GoodDeleteResponse
		s.sendRequest(http.MethodDelete, pathDelete, http.StatusOK, nil, &deleteResponse)

		s.Require().Equal(deleteResponse.ID, createdGood.ID)
		s.Require().Equal(deleteResponse.CampaignID, createdGood.ProjectID)
		s.Require().True(deleteResponse.Removed)
	})
}

func (s *IntegrationTestSuite) TestGetGoods() {
	path := goodsPath + "/create" + "?projectId=1"

	for i := range 20 {
		good := entity.GoodCreateRequest{
			Name: fmt.Sprintf("one_%d", i),
		}

		var createdGood entity.Good

		s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)

		time.Sleep(10 * time.Millisecond)
	}

	s.Run("get goods list successfully with default limit", func() {
		path := "/api/v1/goods/list"
		var response entity.GoodsListResponse
		s.sendRequest(http.MethodGet, path, http.StatusOK, nil, &response)

		s.Require().Len(response.Goods, 10)
		s.Require().Equal(20, response.Meta.Total)
	})

	s.Run("get goods list successfully with custom limit", func() {
		path := fmt.Sprintf("/api/v1/goods/list?limit=%d", 5)
		var response entity.GoodsListResponse
		s.sendRequest(http.MethodGet, path, http.StatusOK, nil, &response)

		s.Require().Len(response.Goods, 5)
		s.Require().Equal(20, response.Meta.Total)
	})

	s.Run("get goods with limit and offset", func() {
		path := fmt.Sprintf("/api/v1/goods/list?limit=%d&offset=%d", 5, 7)
		var response entity.GoodsListResponse
		s.sendRequest(http.MethodGet, path, http.StatusOK, nil, &response)

		s.Require().Len(response.Goods, 5)
		s.Require().Equal(20, response.Meta.Total)
		s.Require().Equal(7, response.Meta.Offset)
	})
}
func (s *IntegrationTestSuite) TestReprioritize() {
	path := goodsPath + "/create" + "?projectId=1"

	var goods []entity.Good
	for i := range 10 {
		good := entity.GoodCreateRequest{
			Name: fmt.Sprintf("one_%d", i),
		}

		var createdGood entity.Good

		s.sendRequest(http.MethodPost, path, http.StatusCreated, &good, &createdGood)
		goods = append(goods, createdGood)

		time.Sleep(10 * time.Millisecond)
	}

	s.Run("reprioritize goods", func() {
		targetGood := goods[2]
		newPriority := 1

		priorityRequest := entity.PriorityRequest{NewPriority: newPriority}

		path := goodsPath + fmt.Sprintf("/reprioritize?id=%d&projectId=%d", targetGood.ID, targetGood.ProjectID)

		var response entity.PriorityResponse
		s.sendRequest(http.MethodPatch, path, http.StatusOK, &priorityRequest, &response)

		s.Require().Len(response.Priorities, 3, "should update target and two affected goods")
		sort.Slice(response.Priorities, func(i, j int) bool {
			return response.Priorities[i].Priority < response.Priorities[j].Priority
		})
		s.Require().Equal(targetGood.ID, response.Priorities[0].ID)
		s.Require().Equal(1, response.Priorities[0].Priority)
	})

	s.Run("reprioritize same priority", func() {
		targetGood := goods[4]
		newPriority := goods[4].Priority

		priorityRequest := entity.PriorityRequest{NewPriority: newPriority}

		path := goodsPath + fmt.Sprintf("/reprioritize?id=%d&projectId=%d", targetGood.ID, targetGood.ProjectID)

		s.sendRequest(http.MethodPatch, path, http.StatusBadRequest, &priorityRequest, nil)
	})

	s.Run("new priority is below zero", func() {
		targetGood := goods[4]
		newPriority := -8

		priorityRequest := entity.PriorityRequest{NewPriority: newPriority}

		path := goodsPath + fmt.Sprintf("/reprioritize?id=%d&projectId=%d", targetGood.ID, targetGood.ProjectID)

		s.sendRequest(http.MethodPatch, path, http.StatusBadRequest, &priorityRequest, nil)
	})
}
