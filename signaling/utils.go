package signaling

import "github.com/pion/randutil"

// RandSeq Util to generate random string with n chars
func RandSeq(n int) string {
	val, err := randutil.GenerateCryptoRandomString(n, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if err != nil {
		panic(err)
	}
	return val
}
