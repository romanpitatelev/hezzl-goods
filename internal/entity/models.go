package entity

type ListRequest struct {
	Sorting    string
	Descending bool
	Limit      int
	Filter     string
	Offset     int
}
