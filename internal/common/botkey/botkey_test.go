package botkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateProducesMatchingKeyPair(t *testing.T) {
	privatePEM, publicPEM, err := Generate()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	pub, err := ParsePublicKey(publicPEM)
	if err != nil {
		t.Fatalf("解析公钥失败: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("公钥长度不正确: %d", len(pub))
	}

	block, _ := pem.Decode(privatePEM)
	if block == nil {
		t.Fatal("私钥不是 PEM")
	}
	if block.Type != "PRIVATE KEY" {
		t.Fatalf("私钥 PEM 类型为 %q", block.Type)
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("解析私钥失败: %v", err)
	}
	priv, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		t.Fatalf("私钥类型为 %T", parsed)
	}

	message := []byte("hello bot")
	signature := ed25519.Sign(priv, message)
	if !ed25519.Verify(pub, message, signature) {
		t.Fatal("生成的私钥与公钥不匹配")
	}
}

func TestParsePublicKeyRejectsRSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 密钥失败: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("编码 RSA 公钥失败: %v", err)
	}
	encoded := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	if _, err := ParsePublicKey(encoded); err == nil {
		t.Fatal("RSA 公钥应当被拒绝")
	}
}

func TestParsePublicKeyRejectsGarbage(t *testing.T) {
	if _, err := ParsePublicKey([]byte("not a pem")); err == nil {
		t.Fatal("非 PEM 内容应当被拒绝")
	}
}

func TestParsePublicKeyRejectsPrivateKey(t *testing.T) {
	privatePEM, _, err := Generate()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	if _, err := ParsePublicKey(privatePEM); err == nil {
		t.Fatal("私钥 PEM 不应当被当作公钥接受")
	}
}

func TestFingerprintStable(t *testing.T) {
	_, publicPEM, err := Generate()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	pub, err := ParsePublicKey(publicPEM)
	if err != nil {
		t.Fatalf("解析公钥失败: %v", err)
	}

	first := Fingerprint(pub)
	second := Fingerprint(pub)
	if first != second {
		t.Fatalf("同一公钥的指纹不一致: %s vs %s", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("SHA-256 指纹长度不正确: %d", len(first))
	}
}
