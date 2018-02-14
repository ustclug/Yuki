package core

import (
	"math/rand"
	"time"
)

// Session contains token, username, and expiration time.
type Session struct {
	Token     string    `bson:"_id"`
	Name      string    `bson:"name"`
	CreatedAt time.Time `bson:"createdAt"`
}

func genToken() string {
	var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	const tokenLength = 20
	b := make([]rune, tokenLength)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// CreateSession creates a new session and returns a token.
func (c *Core) CreateSession(name string) (token string, err error) {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	token = genToken()
	s := Session{
		Token:     token,
		Name:      name,
		CreatedAt: time.Now(),
	}
	err = c.sessColl.With(sess).Insert(s)
	return
}

// LookupToken finds a session according to the given token.
func (c *Core) LookupToken(token string) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	return c.sessColl.With(sess).FindId(token).One(&Session{})
}

// RemoveSession removes the session containing the given token.
func (c *Core) RemoveSession(token string) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	return c.sessColl.With(sess).RemoveId(token)
}
