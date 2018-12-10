package main

import (
	"time"

	"github.com/satori/go.uuid"

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
	now := time.Now()
	active := true
	user := &user{
		Email: email,
		Name:  name,
		Role:  role,
		Pwd:   *hashed,
		Meta: baseMeta{
			MetaCreatedAt: &now,
			MetaUpdatedAt: &now,
			MetaIsActive:  &active,
		},
	}
	userMap, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return nil, err
	}
	// save the user record in dynamo
	if err := putItem(userMap, usersTableName, dbAPI, logger); err != nil {
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

// saveSession
//	* convert the input session item into a dynamodb.AttributeValue map
//	* save the item
func saveSession(sess session, sessionTableName string, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) (*session, error) {
	logger.WithFields(LOGGER.Fields{
		"session":            sess,
		"session_table_name": sessionTableName,
	}).Info("saveSession() - save the incoming session instance into the dynamodb table")
	// check for an id value on the session; if nil, generate a new id & set the meta data
	now := time.Now()
	active := true
	if sess.ID == nil {
		id, _ := uuid.NewV4()
		idVal := id.String()
		sess.ID = &idVal
		sess.Meta = &baseMeta{
			MetaCreatedAt: &now,
			MetaUpdatedAt: &now,
			MetaIsActive:  &active,
		}
	} else {
		// id has a value, update the updated at in meta
		sess.Meta.MetaUpdatedAt = &now
		sess.Meta.MetaIsActive = &active
	}
	// convert to map
	sessMap, err := dynamodbattribute.MarshalMap(sess)
	if err != nil {
		return nil, err
	}
	// save the session
	if err := putItem(sessMap, sessionTableName, dbAPI, logger); err != nil {
		return nil, err
	}
	return &sess, nil
}

// findSessionByID - find a session record in dynamodb by the session id and email associated to the session
func findSessionByID(id, email, sessionTableName string, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) (*session, error) {
	logger.WithFields(LOGGER.Fields{
		"id":                 id,
		"email":              email,
		"session_table_name": sessionTableName,
	}).Info("findSessionByID() - find the session record by the id primary key and email sort key")
	output, err := dbAPI.GetItemRequest(&dynamodb.GetItemInput{
		TableName: aws.String(sessionTableName),
		Key:       map[string]dynamodb.AttributeValue{"id": {S: aws.String(id)}, "email": {S: aws.String(email)}},
	}).Send()
	if err != nil || len(output.Item) == 0 {
		return nil, err
	}
	// unmarshal return into session
	var sess = new(session)
	if err = dynamodbattribute.UnmarshalMap(output.Item, &sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// findSessions - find all session records with the given email sort key
func findSessions(email, sessionTableName string, dbAPI dynamodbiface.DynamoDBAPI, logger *LOGGER.Logger) ([]*session, error) {
	logger.WithFields(LOGGER.Fields{
		"email":              email,
		"session_table_name": sessionTableName,
	}).Info("findSessionByID() - find all session records with the email sort key")
	output, err := dbAPI.QueryRequest(&dynamodb.QueryInput{
		TableName: aws.String(sessionTableName),
		KeyConditions: map[string]dynamodb.Condition{
			"email": {
				ComparisonOperator: dynamodb.ComparisonOperatorEq,
				AttributeValueList: []dynamodb.AttributeValue{{S: aws.String(email)}},
			},
		},
	}).Send()
	if err != nil {
		return nil, err
	}
	if *output.Count == 0 {
		return nil, nil
	}
	var sessions = make([]*session, *output.Count)
	for _, item := range output.Items {
		var sess = new(session)
		if err := dynamodbattribute.UnmarshalMap(item, &sess); err == nil {
			sessions = append(sessions, sess)
		}
	}
	return sessions, nil
}
