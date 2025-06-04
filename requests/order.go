package requests

type CreateOrder struct {
	Comment  string               `json:"comment"`
	Products []CreateOrderProduct `json:"products"`
}

type CreateOrderProduct struct {
	Uuid   string  `json:"uuid"`
	Amount float64 `json:"amount"`
}
