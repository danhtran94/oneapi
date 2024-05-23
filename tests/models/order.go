package models

type Order struct {
	ID int `json:"id"`

	CustomerID int    `json:"customerId"`
	Note       string `json:"note"`

	Items []OrderItem `json:"items"`
}

type OrderItem struct {
	ID       int     `json:"id"`
	OrderID  int     `json:"orderId"`
	Product  Product `json:"product"`
	Quantity int     `json:"quantity"`
	Note     string  `json:"note"`
}

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}
