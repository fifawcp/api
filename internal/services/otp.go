package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

func generateOTP(length int) string {
	const digits = "0123456789"
	otp := make([]byte, length)

	for index := range otp {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		otp[index] = digits[num.Int64()]
	}

	return string(otp)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
