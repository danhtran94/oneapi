package models

type Company struct {
	ID        int    `json:"id" xorm:"pk autoincr"`
	Name      string `json:"name"`
	Employees []User `json:"employees,omitempty"`
}
