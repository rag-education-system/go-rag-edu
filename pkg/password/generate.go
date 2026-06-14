package password

import (
	"crypto/rand"
	"math/big"
)

const defaultPasswordCharset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateRandom(length int) (string, error) {
	if length < 5 {
		length = 8
	}

	result := make([]byte, length)
	max := big.NewInt(int64(len(defaultPasswordCharset)))

	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = defaultPasswordCharset[n.Int64()]
	}

	return string(result), nil
}
