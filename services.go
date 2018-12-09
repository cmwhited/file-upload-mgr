package main

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	LOGGER "github.com/sirupsen/logrus"
)

// findUserByEmail query the users tables to find a user record by the id
func findUserByEmail(email, usersTableName string, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) (*user, error) {
	logger.WithFields(LOGGER.Fields{
		"email":            email,
		"users_table_name": usersTableName,
	}).Info("findUserBydEmail() - attempting to find a user record by the email")
	output, err := dbAPI.GetItemRequest(&dynamodb.GetItemInput{
		TableName: aws.String(usersTableName),
		Key: map[string]dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
	}).Send()
	if err != nil || len(output.Item) == 0 {
		logger.WithFields(LOGGER.Fields{
			"email":            email,
			"users_table_name": usersTableName,
			"error":            err.Error(),
		}).Error("findUserByEmail() - an error occurred while trying to find the user record by email")
		return nil, err
	}
	// unmarshal return into user
	var user = new(user)
	if err = dynamodbattribute.UnmarshalMap(output.Item, &user); err != nil {
		return nil, err
	}
	return user, nil
}

// registerUser register a new user instance using the dynamo service
func registerUser(email, pwd, name, role, usersTableName string, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) (*user, error) {
	logger.WithFields(LOGGER.Fields{
		"email":            email,
		"name":             name,
		"role":             role,
		"users_table_name": usersTableName,
	}).Info("registerUser() - attempting to register a new user")
	hashed, err := hashPwd(pwd)
	if err != nil {
		return nil, err
	}
	// build user instance
	user := &user{
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
		logger.WithFields(LOGGER.Fields{
			"put_item_error": err.Error(),
		}).Error("registerUser() - an error occurred calling the PutItemRequest to store the user")
		return nil, err
	}
	return user, nil
}

// authenticate a user
//	* attempt to find the user with the given email
//		* if not found, return a non-successful authentication
//	* otherwise, validate that the submitted password matches the password on file
//		* if the passwords do not match, return a non-successful authentication
func authenticate(email, pwd, usersTableName string, jwtSecret []byte, tokenExpiryMin int, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) auth {
	user, err := findUserByEmail(email, usersTableName, dbAPI, logger)
	if err != nil {
		return auth{
			Success: false,
			Message: "Unable to find a record with the given email. Please Verify your email and try again",
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
	token, expiry, err := buildToken(user.Email, jwtSecret, tokenExpiryMin)
	if err != nil {
		return auth{
			Success: false,
			Message: err.Error(),
		}
	}
	return auth{Success: true, Token: *token, ExpiresAt: *expiry, User: user}
}
