package crypto

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5(in []byte) string {
	m := md5.Sum(in)
	return hex.EncodeToString(m[:])
}
