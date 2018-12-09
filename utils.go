package main

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// HashPwd hash the input password using the bcrypt lib
func hashPwd(pwd string) (*string, error) {
	password := []byte(pwd) // convert to byte array
	// Use GenerateFromPassword to hash & salt pwd.
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashedPwd := string(hash) // convert returned hashed password to string
	return &hashedPwd, nil
}

// VerifyPwd take the input submitted password and the stored hashed password.
//	- validate that the passwords match
func verifyPwd(hashedPwd, pwd string) bool {
	storedPwd, submittedPwd := []byte(hashedPwd), []byte(pwd)     // convert both the hashed password and submitted password to byte arrays
	err := bcrypt.CompareHashAndPassword(storedPwd, submittedPwd) // compare the password byte slices for equality
	if err != nil {
		return false // passwords do not match, return false
	}
	return true // passwords match, return true
}

// BuildToken build and sign a JWT for the authenticated user.
//	* return the signed token with claims as well as the tokens expiration value
func buildToken(jwtSecret, userID string, tokenExpiryMin int) (*string, *int64, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": userID,
	})
	signedToken, err := token.SignedString(jwtSecret) // sign the token
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()                                                     // get current time
	nowPlusExpiry := now.Add(time.Duration(tokenExpiryMin) * time.Minute) // add 60 minutes to current time to get token expiry
	nowPlusExpiryTimestamp := nowPlusExpiry.UnixNano()                    // get the expiry timestamp
	return &signedToken, &nowPlusExpiryTimestamp, nil
}
