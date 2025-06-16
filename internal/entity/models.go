package entity

import (
	"time"
)

type Project struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type Good struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	Removed     bool      `json:"removed"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GoodCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (g *GoodCreateRequest) Validate() error {
	if g.Name == "" {
		return ErrEmptyName
	}

	return nil
}

type GoodLog struct {
	Operation   string    `json:"operation"`
	GoodID      int       `json:"goodId"`
	ProjectID   int       `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	Removed     bool      `json:"removed"`
	EventTime   time.Time `json:"evenTime"`
}

type GoodUpdate struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type GoodDeleteResponse struct {
	ID         int  `json:"id"`
	CampaignID int  `json:"campaignId"`
	Removed    bool `json:"removed"`
}

type PriorityRequest struct {
	NewPriority int `json:"newPriority"`
}

type Priority struct {
	ID       int `json:"id"`
	Priority int `json:"priority"`
}

type PriorityResponse struct {
	Priorities []Priority `json:"priorities"`
}

type ListRequest struct {
	Limit  int
	Offset int
}

func (l *ListRequest) Validate() {
	if l.Limit <= 0 {
		l.Limit = 10
	}

	if l.Offset < 0 {
		l.Offset = 0
	}
}

type Meta struct {
	Total   int `json:"total"`
	Removed int `json:"removed"`
	Limit   int `json:"limit"`
	Offset  int `json:"offset"`
}

type GoodsListResponse struct {
	Meta  Meta   `json:"meta"`
	Goods []Good `json:"goods"`
}

type URLParams struct {
	ID        int `json:"id"`
	ProjectID int `json:"projectId"`
}
