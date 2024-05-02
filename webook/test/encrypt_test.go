package test

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestPasswordEncrypt(t *testing.T) {
	pwd := []byte("12345&54321")
	encryptPassword, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	err = bcrypt.CompareHashAndPassword(encryptPassword, pwd)
	assert.NoError(t, err)
}
