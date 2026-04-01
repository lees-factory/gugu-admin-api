package product

import "time"

type Clock interface {
	Now() time.Time
}
