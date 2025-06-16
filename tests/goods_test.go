package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

		s.sendRequest(http.MethodPost, path, http.StatusOK, &good, &createdGood)

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

			s.sendRequest(http.MethodPost, path, http.StatusOK, &good, &createdGood)

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
}

func (s *IntegrationTestSuite) TestGetGood() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	good := entity.GoodCreateRequest{
		Name: "test good",
	}

	path := goodsPath + "/create" + "?projectId=1"

	var createdGood entity.Good
	s.sendRequest(http.MethodPost, path, http.StatusOK, &good, &createdGood)

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
	s.sendRequest(http.MethodPost, path, http.StatusOK, &good, &createdGood)

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

func (s *IntegrationTestSuite) TestDeleteGood() {}
func (s *IntegrationTestSuite) TestGetGoods()   {}
func (s *IntegrationTestSuite) TestPrioritize() {}
