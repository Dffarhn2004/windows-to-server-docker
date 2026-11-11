package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

const Window = 120 * time.Second

func Canonical(ts, nonce, hostID string, body []byte) string {
	return ts + "\n" + nonce + "\n" + hostID + "\n" + string(body)
}

func Signature(secret, ts, nonce, hostID string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(Canonical(ts, nonce, hostID, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

func Verify(secret, ts, nonce, hostID, got string, body []byte, now time.Time) error {
	if secret == "" || ts == "" || nonce == "" || hostID == "" || got == "" {
		return errors.New("missing")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("timestamp")
	}
	delta := now.Sub(time.Unix(sec, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > Window {
		return errors.New("expired")
	}
	want := Signature(secret, ts, nonce, hostID, body)
	gb, err1 := hex.DecodeString(got)
	wb, err2 := hex.DecodeString(want)
	if err1 != nil || err2 != nil || len(gb) != len(wb) || subtle.ConstantTimeCompare(gb, wb) != 1 {
		return errors.New("signature")
	}
	return nil
}
