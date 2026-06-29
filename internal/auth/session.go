package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const SessionTTL = 30 * 24 * time.Hour

// Cookie signs an admin id into an HMAC-authenticated token: "<uuid>.<expiryUnix>.<sig>".
type Cookie struct{ secret []byte }

func NewCookie(secret []byte) *Cookie { return &Cookie{secret: secret} }

func (c *Cookie) Sign(adminID uuid.UUID) string {
	exp := time.Now().Add(SessionTTL).Unix()
	return c.signWithExpiry(adminID, exp)
}

func (c *Cookie) signWithExpiry(adminID uuid.UUID, exp int64) string {
	id := adminID.String()
	payload := fmt.Sprintf("%s.%d", id, exp)
	sig := c.mac(payload)
	return payload + "." + sig
}

func (c *Cookie) Verify(token string) (uuid.UUID, bool) {
	// Token format: "<uuid>.<expiryUnix>.<sig>"
	// UUID contains hyphens but no dots; expiryUnix is a plain integer (no dots);
	// so splitting into exactly 3 parts is safe.
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return uuid.Nil, false
	}
	id, expStr, sig := parts[0], parts[1], parts[2]
	payload := id + "." + expStr

	if !hmac.Equal([]byte(sig), []byte(c.mac(payload))) {
		return uuid.Nil, false
	}

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return uuid.Nil, false
	}
	if time.Now().Unix() > exp {
		return uuid.Nil, false
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}

func (c *Cookie) mac(msg string) string {
	m := hmac.New(sha256.New, c.secret)
	m.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
