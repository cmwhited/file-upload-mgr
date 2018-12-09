package main

import (
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/satori/go.uuid"
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

// RegisterUser register a new user instance using the dynamo service
func registerUser(email, pwd, name, role, usersTableName string, dbAPI dynamodbiface.DynamoDBAPI) (*user, error) {
	hashed, err := hashPwd(pwd)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	// build user instance
	user := &user{
		ID:    id.String(),
		Email: email,
		Name:  name,
		Role:  role,
		Pwd:   *hashed,
		Meta: baseMeta{
			MetaCreatedAt: time.Now(),
			MetaUpdatedAt: time.Now(),
			MetaIsActive:  true,
		},
	}
	userMap, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return nil, err
	}
	// save the user record in dynamo
	if _, err := dbAPI.PutItemRequest(&dynamodb.PutItemInput{
		Item:      userMap,
		TableName: aws.String(usersTableName),
	}).Send(); err != nil {
		return nil, err
	}
	return user, nil
}

// Authenticate a user
//	* attempt to find the user with the given email
//		* if not found, return a non-successful authentication
//	* otherwise, validate that the submitted password matches the password on file
//		* if the passwords do not match, return a non-successful authentication
func authenticate(email, pwd, usersTableName, jwtSecret string, tokenExpiryMin int, dbAPI dynamodbiface.DynamoDBAPI) auth {
	output, err := dbAPI.GetItemRequest(&dynamodb.GetItemInput{
		TableName: aws.String(usersTableName),
		Key: map[string]dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
	}).Send()
	if err != nil || len(output.Item) == 0 {
		return auth{
			Success: false,
			Message: "Unable to find a record with the given email. Please Verify your email and try again",
		}
	}
	// unmarshal return into user
	var user = new(user)
	if err = dynamodbattribute.UnmarshalMap(output.Item, &user); err != nil {
		return auth{
			Success: false,
			Message: err.Error(),
		}
	}
	// verify password match
	if !verifyPwd(user.Pwd, pwd) {
		return auth{
			Success: false,
			Message: "The password submitted does not match this users password. Please check the email and password and try again",
		}
	}
	// build the auth token
	token, expiry, err := buildToken(jwtSecret, user.ID, tokenExpiryMin)
	if err != nil {
		return auth{
			Success: false,
			Message: err.Error(),
		}
	}
	return auth{Success: true, Token: *token, ExpiresAt: *expiry}
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
