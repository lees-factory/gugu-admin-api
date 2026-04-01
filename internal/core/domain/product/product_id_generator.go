package product

type IDGenerator interface {
	New() (string, error)
}
