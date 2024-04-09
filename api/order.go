package api

const NormalOrder = 99

type Order interface {
	Order() int
}
