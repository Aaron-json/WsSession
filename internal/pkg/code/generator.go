package code

import (
	"math/rand/v2"
)

const CODE_LENGTH = 5

func Generate() string {
	buf := make([]byte, CODE_LENGTH)
	for i := range len(buf) {
		randNum := rand.IntN(26)
		buf[i] = byte(randNum + 65)
	}
	return string(buf)
}
