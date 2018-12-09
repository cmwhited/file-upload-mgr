package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mitchellh/mapstructure"
	LOGGER "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const bearerTokenKey = "Bearer "

// hashPwd hash the input password using the bcrypt lib
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

// verifyPwd take the input submitted password and the stored hashed password.
//	- validate that the passwords match
func verifyPwd(hashedPwd, pwd string) bool {
	storedPwd, submittedPwd := []byte(hashedPwd), []byte(pwd)     // convert both the hashed password and submitted password to byte arrays
	err := bcrypt.CompareHashAndPassword(storedPwd, submittedPwd) // compare the password byte slices for equality
	if err != nil {
		return false // passwords do not match, return false
	}
	return true // passwords match, return true
}

// buildToken build and sign a JWT for the authenticated user.
//	* return the signed token with claims as well as the tokens expiration value
func buildToken(email string, jwtSecret []byte, tokenExpiryMin int) (*string, *int64, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
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

// validateToken - validate that the incoming Authorization header token is valid:
//		- exists
//		- non-expired
//		- contains the authenticate user email
//	If valid, return the authenticated users email
func validateToken(authHeader interface{}, jwtSecret []byte, logger *LOGGER.Logger) (*string, error) {
	logger.WithFields(LOGGER.Fields{
		"auth_header": authHeader,
	}).Info("validateToken() - validate the incoming authorization header token")
	// validate an Authorization header token is present in the request
	if authHeader == nil {
		return nil, errors.New("no valid Authorization token in request")
	}
	header := authHeader.(string)
	if header == "" {
		return nil, errors.New("no valid Authorization token in request")
	}
	// validate that it is a Bearer token
	if !strings.HasPrefix(header, bearerTokenKey) {
		return nil, errors.New("authorization token is not valid Bearer token")
	}
	t := strings.Replace(header, bearerTokenKey, "", -1)
	// parse the header token
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("there was an parsing the given token. please validate the token is for this service")
		}
		return jwtSecret, nil
	})
	if err != nil {
		logger.WithFields(LOGGER.Fields{
			"auth_header":     authHeader,
			"token":           t,
			"jwt_parse_error": err.Error(),
		}).Error("validateToken() - an error occurred while trying to parse the JWT")
		return nil, err
	}
	// validate token and get claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var decodedToken map[string]string
		err = mapstructure.Decode(claims, &decodedToken)
		if err != nil {
			logger.WithFields(LOGGER.Fields{
				"token":           t,
				"claims":          claims,
				"jwt_parse_error": err.Error(),
			}).Error("validateToken() - an error occurred while trying to get the JWT claims")
			return nil, err
		}
		email := decodedToken["email"]
		return &email, nil
	}
	return nil, errors.New("invalid authorization token") // token is not valid, return error
}
