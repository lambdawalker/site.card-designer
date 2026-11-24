package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"io"
)

var ErrMissing = errors.New("not found")

type Objects interface {
	Get(context.Context, string, int64) ([]byte, error)
	Put(context.Context, string, []byte) error
	Delete(context.Context, string) error
	List(context.Context, string) ([]string, error)
}
type Tokens interface {
	Get(context.Context, string) (Token, error)
	Put(context.Context, Token) error
	Delete(context.Context, string) error
	List(context.Context, string) ([]Token, error)
}
type Token struct {
	ID      string `json:"id"`
	Owner   string `json:"-"`
	Name    string `json:"name"`
	Expires int64  `json:"expires_at"`
}
type S3API interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}
type S3Store struct {
	Client S3API
	Bucket string
}

func (s S3Store) Get(c context.Context, k string, limit int64) ([]byte, error) {
	o, e := s.Client.GetObject(c, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &k})
	if e != nil {
		var ae smithy.APIError
		if errors.As(e, &ae) && (ae.ErrorCode() == "NoSuchKey" || ae.ErrorCode() == "NotFound") {
			return nil, ErrMissing
		}
		return nil, e
	}
	defer o.Body.Close()
	if o.ContentLength != nil && *o.ContentLength > limit {
		return nil, errors.New("object too large")
	}
	b, e := io.ReadAll(io.LimitReader(o.Body, limit+1))
	if int64(len(b)) > limit {
		return nil, errors.New("object too large")
	}
	return b, e
}
func (s S3Store) Put(c context.Context, k string, b []byte) error {
	_, e := s.Client.PutObject(c, &s3.PutObjectInput{Bucket: &s.Bucket, Key: &k, Body: bytes.NewReader(b), ContentType: aws.String("application/json"), ServerSideEncryption: "AES256"})
	return e
}
func (s S3Store) Delete(c context.Context, k string) error {
	_, e := s.Client.DeleteObject(c, &s3.DeleteObjectInput{Bucket: &s.Bucket, Key: &k})
	return e
}
func (s S3Store) List(c context.Context, p string) ([]string, error) {
	out := []string{}
	var next *string
	for {
		o, e := s.Client.ListObjectsV2(c, &s3.ListObjectsV2Input{Bucket: &s.Bucket, Prefix: &p, ContinuationToken: next, MaxKeys: aws.Int32(1000)})
		if e != nil {
			return nil, e
		}
		for _, v := range o.Contents {
			if v.Key != nil {
				out = append(out, *v.Key)
			}
		}
		if len(out) > 1000 {
			return nil, errors.New("too many bundles")
		}
		if !aws.ToBool(o.IsTruncated) {
			return out, nil
		}
		next = o.NextContinuationToken
	}
}

type DynamoAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}
type DynamoTokens struct {
	Client DynamoAPI
	Table  string
}

func key(id string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"token_hash": &types.AttributeValueMemberS{Value: id}}
}
func tokenItem(t Token) map[string]types.AttributeValue {
	b, _ := json.Marshal(struct {
		Owner string
		Token Token
	}{t.Owner, t})
	return map[string]types.AttributeValue{"token_hash": &types.AttributeValueMemberS{Value: t.ID}, "owner": &types.AttributeValueMemberS{Value: t.Owner}, "data": &types.AttributeValueMemberS{Value: string(b)}, "expires_at": &types.AttributeValueMemberN{Value: fmtInt(t.Expires)}}
}
func parseItem(m map[string]types.AttributeValue) (Token, error) {
	v, ok := m["data"].(*types.AttributeValueMemberS)
	if !ok {
		return Token{}, ErrMissing
	}
	var w struct {
		Owner string
		Token Token
	}
	e := json.Unmarshal([]byte(v.Value), &w)
	w.Token.Owner = w.Owner
	return w.Token, e
}
func (d DynamoTokens) Get(c context.Context, id string) (Token, error) {
	o, e := d.Client.GetItem(c, &dynamodb.GetItemInput{TableName: &d.Table, Key: key(id), ConsistentRead: aws.Bool(true)})
	if e != nil {
		return Token{}, e
	}
	return parseItem(o.Item)
}
func (d DynamoTokens) Put(c context.Context, t Token) error {
	_, e := d.Client.PutItem(c, &dynamodb.PutItemInput{TableName: &d.Table, Item: tokenItem(t), ConditionExpression: aws.String("attribute_not_exists(token_hash)")})
	return e
}
func (d DynamoTokens) Delete(c context.Context, id string) error {
	_, e := d.Client.DeleteItem(c, &dynamodb.DeleteItemInput{TableName: &d.Table, Key: key(id)})
	return e
}
func (d DynamoTokens) List(c context.Context, owner string) ([]Token, error) {
	out := []Token{}
	var last map[string]types.AttributeValue
	for {
		o, e := d.Client.Query(c, &dynamodb.QueryInput{TableName: &d.Table, IndexName: aws.String("owner-index"), ExclusiveStartKey: last, KeyConditionExpression: aws.String("#o = :o"), ExpressionAttributeNames: map[string]string{"#o": "owner"}, ExpressionAttributeValues: map[string]types.AttributeValue{":o": &types.AttributeValueMemberS{Value: owner}}})
		if e != nil {
			return nil, e
		}
		for _, m := range o.Items {
			t, e := parseItem(m)
			if e != nil {
				return nil, e
			}
			out = append(out, t)
		}
		if len(o.LastEvaluatedKey) == 0 {
			return out, nil
		}
		last = o.LastEvaluatedKey
	}
}
