// Package botkey 负责机器人 Ed25519 密钥的生成、解析与指纹计算。
package botkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	privatePEMType = "PRIVATE KEY"
	publicPEMType  = "PUBLIC KEY"
)

// Generate 生成 Ed25519 密钥对，返回 PKCS#8 私钥与 PKIX 公钥的 PEM 编码。
func Generate() (privatePEM, publicPEM []byte, err error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("生成 Ed25519 密钥失败: %w", err)
	}

	privateDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("编码私钥失败: %w", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return nil, nil, fmt.Errorf("编码公钥失败: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: privatePEMType, Bytes: privateDER}),
		pem.EncodeToMemory(&pem.Block{Type: publicPEMType, Bytes: publicDER}), nil
}

// ParsePublicKey 解析 PKIX PEM 公钥，并确认它是 Ed25519 公钥。
func ParsePublicKey(data []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("不是有效的 PEM 文件")
	}
	if block.Type != publicPEMType {
		return nil, fmt.Errorf("PEM 类型为 %q，期望 %q", block.Type, publicPEMType)
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败: %w", err)
	}

	pub, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("公钥类型为 %T，不是 Ed25519", parsed)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("Ed25519 公钥长度为 %d，期望 %d", len(pub), ed25519.PublicKeySize)
	}

	return pub, nil
}

// Fingerprint 返回公钥的 SHA-256 十六进制指纹。
func Fingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:])
}
