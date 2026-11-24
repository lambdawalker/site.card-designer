package api

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"net/http"
	"os"
	"time"
)

func New(ctx context.Context) (http.Handler, error) {
	c := Config{os.Getenv("COGNITO_DOMAIN"), os.Getenv("COGNITO_CLIENT_ID"), os.Getenv("REDIRECT_URI"), os.Getenv("LOGOUT_URI"), os.Getenv("AWS_REGION"), os.Getenv("COGNITO_USER_POOL_ID")}
	bucket, table := os.Getenv("DATA_BUCKET"), os.Getenv("TOKEN_TABLE")
	if bucket == "" || table == "" || c.Region == "" || c.ClientID == "" || c.UserPoolID == "" {
		return nil, errors.New("DATA_BUCKET, TOKEN_TABLE, AWS_REGION, COGNITO_CLIENT_ID and COGNITO_USER_POOL_ID required")
	}
	cfg, e := config.LoadDefaultConfig(ctx, config.WithRegion(c.Region))
	if e != nil {
		return nil, e
	}
	objects := S3Store{s3.NewFromConfig(cfg), bucket}
	tokens := DynamoTokens{dynamodb.NewFromConfig(cfg), table}
	issuer := "https://cognito-idp." + c.Region + ".amazonaws.com/" + c.UserPoolID
	auth := &Auth{Issuer: issuer, ClientID: c.ClientID, Tokens: tokens, HTTP: &http.Client{Timeout: 10 * time.Second}}
	return &Server{Config: c, Auth: auth, Objects: objects, Tokens: tokens, HF: &HF{Objects: objects, Token: os.Getenv("HF_TOKEN")}}, nil
}
