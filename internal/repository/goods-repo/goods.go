package goodsrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/postgres"
)

type Repo struct {
	db *postgres.DataStore
}

func New(db *postgres.DataStore) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) CreateGood(ctx context.Context, projectID int, req entity.GoodCreateRequest) (entity.Good, error) {
	var (
		good     entity.Good
		priority int
	)

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx postgres.Transaction) error {
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

func (r *Repo) GetGood(ctx context.Context, id int, projectID int) (entity.Good, error) {
	var good entity.Good

	query := `
SELECT id, project_id, name, COALESCE(description, ''), priority, removed, created_at
FROM goods
WHERE id = $1 AND project_id = $2
	
`
	row := r.db.GetTXFromContext(ctx).QueryRow(ctx, query, id, projectID)

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
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Good{}, entity.ErrGoodNotFound
		}

		return entity.Good{}, fmt.Errorf("failed to scan good: %w", err)
	}

	return good, nil
}

func (r *Repo) UpdateGood(ctx context.Context, id int, projectID int, goodUpdate entity.GoodUpdate) (entity.Good, error) {
	var good entity.Good

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx postgres.Transaction) error {
		queryCheck := `
SELECT id FROM goods WHERE id = $1 AND project_id = $2 FOR UPDATE
`
		row := tx.QueryRow(ctx, queryCheck, id, projectID)

		var goodID int
		if err := row.Scan(&goodID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return entity.ErrGoodNotFound
			}

			return fmt.Errorf("failed to check good's existence in the db: %w", err)
		}

		queryUpdate := `
UPDATE goods
SET name = $1,
	description = COALESCE($2, description)
WHERE id = $3 AND project_id = $4
RETURNING id, project_id, name, COALESCE(description, ''), priority, removed, created_at)		
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
			return entity.Good{}, fmt.Errorf("good is not found in UpdateGood(): %w", err)
		}

		return entity.Good{}, fmt.Errorf("failed to update good: %w", err)
	}

	return good, nil
}

func (r *Repo) DeleteGood(ctx context.Context, id int, projectID int) (entity.GoodDeleteResponse, error) {
	var response entity.GoodDeleteResponse

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx postgres.Transaction) error {
		queryCheck := `
SELECT id FROM goods WHERE id = $1 AND project_id = $2 FOR UPDATE
`
		row := tx.QueryRow(ctx, queryCheck, id, projectID)

		var goodID int
		if err := row.Scan(&goodID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return entity.ErrGoodNotFound
			}

			return fmt.Errorf("failed to check good's existence: %w", err)
		}

		queryDelete := `
UPDATE goods
SET removed = true
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, removed
`
		row = tx.QueryRow(ctx, queryDelete, id, projectID)

		err := row.Scan(
			&response.ID,
			&response.CampaignID,
			&response.Removed)
		if err != nil {
			return fmt.Errorf("failed to scan deleted good: %w", err)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, entity.ErrGoodNotFound) {
			return entity.GoodDeleteResponse{}, fmt.Errorf("good is not found in DeleteGood(): %w", err)
		}

		return entity.GoodDeleteResponse{}, fmt.Errorf("failed to delete good: %w", err)
	}

	return response, nil
}

func (r *Repo) GetGoods(ctx context.Context, request entity.ListRequest) ([]entity.Good, entity.Meta, error) {
	var (
		meta  entity.Meta
		goods []entity.Good
	)

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx postgres.Transaction) error {
		row := tx.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE removed = true) FROM goods`)

		err := row.Scan(&meta.Total, &meta.Removed)
		if err != nil {
			return fmt.Errorf("failed to get goods count: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, project_id, name, COALESCE(description, ''), priority, removed, created_at
			FROM goods
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`,
			request.Limit, request.Offset,
		)
		if err != nil {
			return fmt.Errorf("error while quering in GetGoods(): %w", err)
		}

		defer rows.Close()

		for rows.Next() {
			var good entity.Good
			if err := rows.Scan(
				&good.ID,
				&good.ProjectID,
				&good.Name,
				&good.Description,
				&good.Priority,
				&good.Removed,
				&good.CreatedAt,
			); err != nil {
				return fmt.Errorf("error scanning good: %w", err)
			}

			goods = append(goods, good)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, entity.Meta{}, fmt.Errorf("failed to get goods: %w", err)
	}

	meta.Limit = request.Limit
	meta.Offset = request.Offset

	return goods, meta, nil
}

//nolint:funlen
func (r *Repo) Reprioritize(ctx context.Context, id int, projectID int, newPriority entity.PriorityRequest) ([]entity.Priority, error) {
	var updatedPriorities []entity.Priority

	err := r.db.WithinTransaction(ctx, func(ctx context.Context, tx postgres.Transaction) error {
		currentPriority, err := r.getCurrentPriority(ctx, tx, id, projectID)
		if err != nil {
			return fmt.Errorf("error getting current priority: %w", err)
		}

		if currentPriority == newPriority.NewPriority {
			updatedPriorities = append(updatedPriorities, entity.Priority{
				ID:       id,
				Priority: newPriority.NewPriority,
			})

			return nil
		}

		var updateQuery string
		if newPriority.NewPriority < currentPriority {
			updateQuery = `
UPDATE goods
SET priority = priority + 1
WHERE priority > $1 AND priority <= $2
RETURNING id, priority
`
		}

		rows, err := tx.Query(ctx, updateQuery, newPriority.NewPriority, currentPriority)
		if err != nil {
			return fmt.Errorf("failed to update priorities: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p entity.Priority
			if err := rows.Scan(&p.ID, &p.Priority); err != nil {
				return fmt.Errorf("failed to scan updated priority: %w", err)
			}

			updatedPriorities = append(updatedPriorities, p)
		}

		_, err = tx.Exec(ctx, `
			UPDATE goods
			SET priority = $1
			WHERE id = $2 AND project_id = $3`,
			newPriority.NewPriority, id, projectID)
		if err != nil {
			return fmt.Errorf("failed to update target priority: %w", err)
		}

		updatedPriorities = append(updatedPriorities, entity.Priority{
			ID:       id,
			Priority: newPriority.NewPriority,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reprioritize: %w", err)
	}

	return updatedPriorities, nil
}

func (r *Repo) getCurrentPriority(ctx context.Context, tx postgres.Transaction, id, projectID int) (int, error) {
	var currentPriority int

	row := tx.QueryRow(ctx,
		`SELECT priority FROM goods
		WHERE id = $1 AND project_id = $2
		FOR UPDATE`,
		id, projectID,
	)

	if err := row.Scan(&currentPriority); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, entity.ErrGoodNotFound
		}

		return 0, fmt.Errorf("failed to get current priority: %w", err)
	}

	return currentPriority, nil
}
