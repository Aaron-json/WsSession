package pool

type PoolError int

const (
	DUPLICATE_KEY PoolError = 0
	MAX_CAPACITY  PoolError = 1
	KEY_NOT_FOUND PoolError = 2
)

func (e PoolError) String() string {
	switch e {
	case DUPLICATE_KEY:
		return "Key already exists"
	case MAX_CAPACITY:
		return "The pool is full"
	case KEY_NOT_FOUND:
		return "Key does not exist in the pool"
	default:
		return "Unspecified Error"
	}
}

func (e PoolError) Error() string {
	return e.String()
}
