package util

import "testing"

func TestEncryptAndDecryptAES128(t *testing.T) {
	key := "example key 1234"
	data := "Hello World"
	encryptedData := AESEncrypt(key, []byte(data))
	if string(AESDecrypt(key, encryptedData)) != data {
		t.Error("encrypt error")
	}
}

func TestEncryptAndDecryptAES192(t *testing.T) {
	key := "example key 123412345678"
	data := "Hello World"
	encryptedData := AESEncrypt(key, []byte(data))
	if string(AESDecrypt(key, encryptedData)) != data {
		t.Error("encrypt error")
	}
}

func TestRSAEncryptAndDecrpt(t *testing.T) {
	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQC4frqehpBf6ewGU0VAU7hHDO4dkBzDLd0SsJp5KGtZRYqFlXJG
cgUw4Rpskb0T8uPhd2yambJ1IEbX16wkqq3jF8Hgs4WIfCyhYti80Lfi/GpnzuMn
0dyn5H5QT2FP9KXDSKsZcCm1qzEV1W9SS7DbsZYUD+Lx23QyfP/3281mQwIDAQAB
AoGAfodbUXET/tOc7VGagt1n2kKB44B8WVdQ8Ipxxnnz9Ut+DtNJhgqYiMc4qhDh
TZcctfqDXxvdifpS26CsDJGJorTabNSn+YibuOyFe5MFMXKrb/cB/PfPjJfMjRfG
O5g2xiLMqcK4MXSREKrmdlQ7g2Ysh8NhyKvjdr8gS4elgiECQQDg0d96H7DJVI3v
Z3R4ICOe0KV0+JRcFAa90cFOU8odSrafKhQaznEYqSlKKotmTLNceP9JgDo1BJZW
1ZLYD5mpAkEA0hUkiBLvPc0Y5/Ow/CtphTzhj68oioZLd+N4wTgP5EKUIbO1p10i
k3Ng0RSViHAb+XsvrppUgyITfnGB9ersCwJBAJWrYum8m0cNYYiWCTXHv68FHIGo
06wRMQPB1r08jvu9N6Lysnu+IBDY3UIg3Lj4KxhO/TWDhjyxlxysBpyMljECQQCp
C4hE0m+eXC3hX08X6trS8qVSGBDYPq31f53IZJMtCoHmCJRwYtoSqjHKq/STQBrS
ilRY/ChrCH2FLlL0Dh/3AkEA2KAcZRt/u8lxOj0dWZF4nUoTdFqZGyzRJPskaO7j
tI4zQh6D9noVFujVMvtMPtR6CCaT9Co+42Li/h1WcMnBsg==
-----END RSA PRIVATE KEY-----`)
	publicKey := []byte(`-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC4frqehpBf6ewGU0VAU7hHDO4d
kBzDLd0SsJp5KGtZRYqFlXJGcgUw4Rpskb0T8uPhd2yambJ1IEbX16wkqq3jF8Hg
s4WIfCyhYti80Lfi/GpnzuMn0dyn5H5QT2FP9KXDSKsZcCm1qzEV1W9SS7DbsZYU
D+Lx23QyfP/3281mQwIDAQAB
-----END PUBLIC KEY-----`)
	testData := []byte("Hello World")
	ecnryptedData, err := RSAPublicKeyEncrypt(&publicKey, &testData)
	if err != nil {
		t.Error(err.Error())
		return
	}
	readData, err := RsaPrivateKeyDecrypt(&privateKey, &ecnryptedData)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if string(readData) != string(testData) {
		t.Error("data not match")
	}
}
