package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/lambdawalker/site.card-designer/backend/internal/api"
	"log"
)

func main() {
	h, e := api.New(context.Background())
	if e != nil {
		log.Fatal(e)
	}
	lambda.Start(httpadapter.NewV2(h).ProxyWithContext)
}
