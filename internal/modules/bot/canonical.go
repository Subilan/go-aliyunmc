package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Canonical 返回 v1 请求签名使用的规范化字符串。
// 使用请求头里的原始字符串，末尾不带换行。
func Canonical(method, requestURI, keyID, timestamp, nonce string, body []byte) string {
	sum := sha256.Sum256(body)

	return strings.Join([]string{
		"v1",
		keyID,
		method,
		requestURI,
		timestamp,
		nonce,
		hex.EncodeToString(sum[:]),
	}, "\n")
}
