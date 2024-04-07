package code

import (
	"math/rand/v2"
)

func Generate(length int) string {
	buf := make([]byte, length)
	for i := range len(buf) {
		randNum := rand.IntN(26)
		buf[i] = byte(randNum + 65)
	}
	return string(buf)
}
