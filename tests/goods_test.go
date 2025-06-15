package tests

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

func (s *IntegrationTestSuite) TestGetGood()    {}
func (s *IntegrationTestSuite) TestUpdateGood() {}
func (s *IntegrationTestSuite) TestDeleteGood() {}
func (s *IntegrationTestSuite) TestGetGoods()   {}
func (s *IntegrationTestSuite) TestPrioritize() {}
