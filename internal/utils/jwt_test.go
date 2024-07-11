package utils_test

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

const (
	pubkey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAu+s1YTUZXA4BMIoqPPgd
F2ll9wek/rOssGYA65w6oEI6FmJqo1ESEXy+b0RdgwY72nAANlKd+/uZ8PQ+u42z
KT0JhEJK9F88gUiFacI6s/wP1gutWoTkQl7lqmEWCX3U9f4ZJOs+yOajQoHQ8Sl3
yM+vsrSlWMIpHqxx2mU+FytfqX9nEHgTzuGxOXhveLtidoVRauAZwSSQQGB4c3uS
aP7VSlTF290Q5PxhuS0LmI8QEypq6Dd0fLZn+jMWa5CE9mSGDwpqSWTL5DoNI1Sc
IxNPzDLByRR8nXfWrKZiA4yq9EtK6qtZLnayWiNRGK+gtUTsnF2Q11IhkHuuIaRI
sG+CskDpbrbFXRjFmjxfQZzdS3/xCzFEOF/N4g+9vm0lx9T/Qm1XWap+LwJXQnmO
Lhjnxflm6Ttnq9R0nuKE1GB1D3zWIppf8XAx+LdarTqEcXbbHRVReyGOi6sNYNVf
REWfNbDrBwBUo9/GsyInzM1pFNg9cG2ILyEZuSqyp5lMJMrM45u80/w0ylGS7jZA
vZwN4TZLW1sJTN8Kv7RPMOfzo+s8DvKv+Jpkek7SQJ9tTVnsuzStj8AewoZJO/2r
nJEXREIEYBnkr+2HLbFQ0PHJmaIcGfcWEXHFXPHGn7/gamN6D0K9ornRJ4Rlla+y
3fKg8rhRNniRN0pOO3wv4FECAwEAAQ==
-----END PUBLIC KEY-----`
	nonAssociatedPubkey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAsRfCiFJrtBBoIJgdwedC
ew6teYrHLzDlJ0ta3//YTwDMQF2tIufni96cf78KUBAtHUgSyYfrtyepV1E9iJ7P
eexQayYbwQscrWX4WM+9b43jhRVvzVCMw/RpkFI/NJuOKf4vSB9JZtOCXDNwTv7s
45GP9qQK3GITa6TtlIv7t7gWBsVvFaCrHKj1/k18K09aspc3hnGI0KVIWxr2JOZc
GvZYHMvYbTHzWB74mUIHBHxd0RoxKJ15aoxd6Q+ufYceb2ykBiN2zZFHzEYU1t3+
bXv1iHz51IGtC/Q2BXDNsbXVuOuSxcOEyLFaU1X3rFdbdKr3gZGUhUzaWsCtwHgA
rNU2Qd9/PjJXQhSW/FIExSyjgO2GNL9+J/i93T1jpeDqHS/pLrYYqsiJ8sPmvnYI
VTfP6c790ctK2Et2lvRDlP1LEwZhm8h64bzFhxGhqH+J3GLKabGsMbl0nZvefB+7
ZRGjZeD/s7HrtbW0EWP2aLjG2U/KetWUN/nZO/718nYFh9LjJwdwol7ZjLBFJw1a
97aIgRDl41uORgyLieL7iGGPY9eLfNCU37u++VXc1MrUKgCMC+Ee+MkImxZ6RTMe
dxWHtQPRWjZOgPRyYXX5u8yRjya0/Lt0COlzuaWRm5IksVUrYUqz4Q5efeU1/0kN
tj9ue+ao4wtoE2lGIGVJGxsCAwEAAQ==
-----END PUBLIC KEY-----`
	privkey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
	nonAssociatedPrivkey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
)

func TestJWT(t *testing.T) {
	t.Parallel()

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, utils.DeviceJWT{
		Identity: "test",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		},
	})
	blk, _ := pem.Decode([]byte(privkey))
	key, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = utils.VerifyDeviceJWT("test", pubkey, tokenStr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	blk, _ = pem.Decode([]byte(nonAssociatedPrivkey))
	key, err = x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	tokenStr, err = token.SignedString(key)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = utils.VerifyDeviceJWT("test", pubkey, tokenStr)
	if err == nil {
		t.Error("expected error, got nil")
	}

	blk, _ = pem.Decode([]byte(privkey))
	key, err = x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	tokenStr, err = token.SignedString(key)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = utils.VerifyDeviceJWT("test", nonAssociatedPubkey, tokenStr)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGenerateVerify(t *testing.T) {
	t.Parallel()

	secret := "changeme"
	uid := uint(1)
	token, err := utils.GenerateJWT(secret, uid)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	gotUID, err := utils.VerifyJWT(secret, token)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if gotUID != uid {
		t.Errorf("expected %d, got %d", uid, gotUID)
	}
}
