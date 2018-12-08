package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	query := `{hello}`
	p := params{Query: query}
	rJSON, _ := json.Marshal(p)
	request := events.APIGatewayProxyRequest{
		Body:       string(rJSON),
		HTTPMethod: "post",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	ctx := context.Background()
	expectedResponse := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"hello":"World"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	response, err := Handler(ctx, request)

	assert.Contains(t, response.Body, expectedResponse.Body)
	assert.Equal(t, response.Headers, expectedResponse.Headers)
	assert.Equal(t, err, nil)

}
