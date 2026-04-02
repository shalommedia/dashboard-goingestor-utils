package s3client

type UploadInput struct {
	Bucket      string
	Key         string
	Body        []byte
	ContentType string
}
