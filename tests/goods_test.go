package tests

import (
	"net/http"

	"github.com/romanpitatelev/hezzl-goods/internal/entity"
)

func (s *IntegrationTestSuite) TestCreateGood() {
	good := entity.GoodCreateRequest{
		Name: "one",
	}

	s.Run("create good successfully", func() {
		path := goodsPath + "/create" + "?projectId=1"

		var createdGood entity.Good
		s.sendRequest(http.MethodPost, path, http.StatusNotFound, &good, &createdGood)

		s.Require().Equal(1, createdGood.ID)
		s.Require().Equal(1, createdGood.ProjectID)
		s.Require().Equal(good.Name, createdGood.Name)
		s.Require().Equal(1, createdGood.Priority)
		s.Require().Equal(false, createdGood.Removed)
	})
}
func (s *IntegrationTestSuite) TestGetGood()    {}
func (s *IntegrationTestSuite) TestUpdateGood() {}
func (s *IntegrationTestSuite) TestDeleteGood() {}
func (s *IntegrationTestSuite) TestGetGoods()   {}
func (s *IntegrationTestSuite) TestPrioritize() {}
