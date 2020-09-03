package session

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// SessionCookieSecret is used to encrypt the cookie session id.
	SessionCookieSecret = ""
)

const (
	sessionLifeInSec = 7200
	cookieLifeInSec  = 7200
	cookieName       = "session"
)

// Session stores necessary information for a telehealth session.
type Session struct {
	ID        string
	ExpiresAt time.Time
	Value     map[string]interface{}
}

// Put stores the key-val pair in Session.
func (s *Session) Put(key string, val interface{}) {
	if s.Value == nil {
		s.Value = make(map[string]interface{})
	}
	s.Value[key] = val
}

// Get returns the value for the given key.
func (s *Session) Get(key string) interface{} {
	if s.Value == nil {
		return nil
	}
	v, ok := s.Value[key]
	if !ok {
		return nil
	}
	return v
}

// New creates a new session and set cookie containning the encoded session id.
func New(m *StoreManager, w http.ResponseWriter, r *http.Request) (*Session, error) {
	expireAt := time.Now().Add(sessionLifeInSec * time.Second)
	s, err := m.Create(expireAt)
	if err != nil {
		return nil, err
	}
	expiration := time.Now().Add(cookieLifeInSec * time.Second)
	cookie := &http.Cookie{Name: cookieName, Value: encodeSessionID(s.ID), Expires: expiration}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return s, nil
}

// Find returns the session whose id matches the session id in the cookie of the request.
func Find(m *StoreManager, r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}
	sid, err := decodeSessionID(cookie.Value)
	if err != nil {
		return nil, err
	}

	return m.Find(sid)
}

func encodeSessionID(sid string) string {
	b := base64.StdEncoding.EncodeToString([]byte(sid))
	s := fmt.Sprintf("%s-%s", b, signature(sid))
	return url.QueryEscape(s)
}

func signature(sid string) string {
	h := hmac.New(sha1.New, []byte(SessionCookieSecret))
	h.Write([]byte(sid))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func decodeSessionID(esid string) (string, error) {
	esid, err := url.QueryUnescape(esid)
	if err != nil {
		return "", err
	}

	vals := strings.Split(esid, "-")
	if len(vals) != 2 {
		return "", fmt.Errorf("Invalid session ID")
	}

	bsid, err := base64.StdEncoding.DecodeString(vals[0])
	if err != nil {
		return "", err
	}
	sid := string(bsid)

	sig := signature(sid)
	if sig != vals[1] {
		return "", fmt.Errorf("Invalid session ID")
	}
	return sid, nil
}
