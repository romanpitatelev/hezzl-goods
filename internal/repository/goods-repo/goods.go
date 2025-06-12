package goodsrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *Repo) GetGood(ctx context.Context, id int, projectID int) (entity.Good, error)

func (r *Repo) UpdateGood(ctx context.Context, id int, projectID int, goodUpdate entity.GoodUpdate) (entity.Good, error) {
	var good entity.Good

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx store.Transaction) error {
		queryCheck := `
SELECT id FROM goods WHERE id = $1 AND sproject_id = $2 FOR UPDATE
`
		row := tx.QueryRow(ctx, queryCheck, id, projectID)

		var goodID int
		if err := row.Scan(&goodID); err != nil {
			if err == pgx.ErrNoRows {
				return entity.ErrGoodNotFound
			}
			return fmt.Errorf("failed to check good's existence in the db: %w", err)
		}

		queryUpdate := `
UPDATE goods
SET name = $1,
	description = COALESCE($2, description)
WHERE id = $3 AND project_id = $4
RETURNING id, project_id, name, COALESCE(description, '', priority, removed, created_at)		
`
		row = tx.QueryRow(ctx, queryUpdate,
			goodUpdate.Name,
			goodUpdate.Description,
			id,
			projectID,
		)

		err := row.Scan(
			&good.ID,
			&good.ProjectID,
			&good.Name,
			&good.Description,
			&good.Priority,
			&good.Removed,
			&good.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan updated good: %w", err)
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, entity.ErrGoodNotFound) {
			return entity.Good{}, err
		}
		return entity.Good{}, fmt.Errorf("failed to update good: %w", err)
	}

	return good, nil
}

func (r *Repo) DeleteGood(ctx context.Context, id int, project_id int) (entity.GoodDeleteResponse, error)
func (r *Repo) GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, error)
func (r *Repo) Reprioritize(ctx context.Context, id int, project_id int, new_priority entity.PriorityRequest) (entity.PriorityResponse, error)
