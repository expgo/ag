package api

//go:generate ag

/*
Order

	@Enum {
		First = 0
		High = 10
		AboveNormal = 20
		Normal = 30
		BelowNormal = 40
	}
*/
type Order int

type IOrder interface {
	Order() Order
}
