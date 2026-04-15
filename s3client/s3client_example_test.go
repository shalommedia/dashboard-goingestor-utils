package s3client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type capturePutObjectClient struct {
	input *s3.PutObjectInput
}

func (c *capturePutObjectClient) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	c.input = params
	return &s3.PutObjectOutput{}, nil
}

func ExamplePutObjectStream() {
	client := &capturePutObjectClient{}
	data := "1,alice@example.com\n2,bob@example.com\n"

	err := putObjectStream(context.Background(), client, "sync-bucket", "contacts.csv", int64(len(data)), "text/csv", strings.NewReader(data))
	if err != nil {
		fmt.Println("upload failed")
		return
	}

	uploaded, _ := io.ReadAll(client.input.Body)
	fmt.Printf("bucket=%s key=%s bytes=%d\n", *client.input.Bucket, *client.input.Key, len(uploaded))
	// Output: bucket=sync-bucket key=contacts.csv bytes=38
}
