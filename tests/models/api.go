package models

// Collection is a generic collection type
//
// @oas: kind=response placeholder=T name=%sCollection
type Collection[T any] struct {
	Total int `json:"total"`
	Items []T `json:"items"`
}
