package goodshandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/romanpitatelev/hezzl-goods/internal/controller/rest/common"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
)

type goodsService interface {
	CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error)
	GetGood(ctx context.Context, id int, projectID int) (entity.Good, error)
	UpdateGood(ctx context.Context, id int, projectID int, goodUpdate entity.GoodUpdate) (entity.Good, error)
	DeleteGood(ctx context.Context, id int, projectID int) (entity.GoodDeleteResponse, error)
	GetGoods(ctx context.Context, request entity.ListRequest) (entity.GoodsListResponse, error)
	Reprioritize(ctx context.Context, id int, projectID int, newPriority entity.PriorityRequest) (entity.PriorityResponse, error)
}

type Handler struct {
	goodsService goodsService
}

func New(goodsService goodsService) *Handler {
	return &Handler{
		goodsService: goodsService,
	}
}

func (h *Handler) CreateGood(w http.ResponseWriter, r *http.Request) {
	param := r.URL.Query().Get("projectId")
	projectID, err := strconv.Atoi(param)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	var req entity.GoodCreateRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "error decoding request body", http.StatusBadRequest)

		return
	}

	ctx := r.Context()

	createdGood, err := h.goodsService.CreateGood(ctx, projectID, req)
	if err != nil {
		common.ErrorResponse(w, "error creating good", err)

		return
	}

	common.OkResponse(w, http.StatusOK, createdGood)
}

func (h *Handler) GetGood(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) UpdateGood(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	idStr := queryParams.Get("id")
	projectIDStr := queryParams.Get("projectId")
	if idStr == "" || projectIDStr == "" {
		http.Error(w, "empty good id or project id", http.StatusBadRequest)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	var req entity.GoodUpdate
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "error decoding request body", http.StatusBadRequest)

		return
	}

	ctx := r.Context()

	updatedGood, err := h.goodsService.UpdateGood(ctx, id, projectID, req)
	if err != nil {
		common.ErrorResponse(w, "error creating good", err)

		return
	}

	common.OkResponse(w, http.StatusOK, updatedGood)
}

func (h *Handler) DeleteGood(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	idStr := queryParams.Get("id")
	projectIDStr := queryParams.Get("projectId")
	if idStr == "" || projectIDStr == "" {
		http.Error(w, "empty good id or project id", http.StatusBadRequest)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	ctx := r.Context()

	deletedGood, err := h.goodsService.DeleteGood(ctx, id, projectID)
	if err != nil {
		common.ErrorResponse(w, "error deleting good", err)

		return
	}

	common.OkResponse(w, http.StatusOK, deletedGood)
}

func (h *Handler) GetGoods(w http.ResponseWriter, r *http.Request) {
	request := common.GetListRequest(r)

	if request.Limit <= 0 {
		request.Limit = 10
	}

	if request.Offset < 0 {
		request.Offset = 0
	}

	ctx := r.Context()

	goods, err := h.goodsService.GetGoods(ctx, request)
	if err != nil {
		common.ErrorResponse(w, "error listing goods", err)

		return
	}

	common.OkResponse(w, http.StatusOK, goods)
}

func (h *Handler) Reprioritize(w http.ResponseWriter, r *http.Request) {}
