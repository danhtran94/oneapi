package models

type Company struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Employees []User `json:"employees"`
}
