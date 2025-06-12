package entity

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type UserID = uuid.UUID

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
	EventTime   time.Time `json:"event_time"`
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

type PriorityResponse struct {
}

type Claims struct {
	UserID UserID  `json:"userId"`
	Email  *string `json:"email"`
	Role   *string `json:"role"`
	jwt.RegisteredClaims
}

type UserInfo struct {
	ID    UserID `json:"userId"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ListRequest struct {
	Sorting    string
	Descending bool
	Limit      int
	Filter     string
	Offset     int
}
