package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
)

type logger struct {
	lambda.Handler
}

func (l *logger) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	fmt.Println(string(payload))
	resp, err := l.Handler.Invoke(ctx, payload)
	fmt.Println(string(resp))
	return resp, err
}
