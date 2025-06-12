package goodsrepo

import (
	"context"
	"fmt"

	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/store"
)

type Repo struct {
	db *store.DataStore
}

func New(db *store.DataStore) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error) {
	var good entity.Good
	var priority int

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx store.Transaction) error {
		row := tx.QueryRow(ctx, `SELECT COALESCE(MAX(priority), 0) + 1 FROM goods`)
		if err := row.Scan(&priority); err != nil {
			return fmt.Errorf("failed to get max priority: %w", err)
		}

		query := `
INSERT INTO goods (project_id, name, description, priority)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, name, COALESCE(description, ''), priority, removed, created_at	
`

		goodRow := tx.QueryRow(ctx, query, projectID, req.Name, req.Description, priority)

		err := goodRow.Scan(
			&good.ID,
			&good.ProjectID,
			&good.Name,
			&good.Description,
			&good.Priority,
			&good.Removed,
			&good.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan good: %w", err)
		}

		return nil
	})
	if err != nil {
		return entity.Good{}, fmt.Errorf("failed to create good: %w", err)
	}

	return good, nil
}

func (r *Repo) GetGood(ctx context.Context, id int, project_id int) (entity.Good, error)
func (r *Repo) UpdateGood(ctx context.Context, id int, project_id int, goodUpdate entity.GoodUpdate) (entity.Good, error)
func (r *Repo) DeleteGood(ctx context.Context, id int, project_id int) (entity.GoodDeleteResponse, error)
func (r *Repo) GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, error)
func (r *Repo) Reprioritize(ctx context.Context, id int, project_id int, new_priority entity.PriorityRequest) (entity.PriorityResponse, error)
