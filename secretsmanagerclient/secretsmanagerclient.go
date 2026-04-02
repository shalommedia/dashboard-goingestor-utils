package secretsmanagerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var (
	defaultClient *secretsmanager.Client
	defaultErr    error
	once          sync.Once
)

type getSecretValueAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// New creates a Secrets Manager client from the ambient AWS configuration.
func New(ctx context.Context) (*secretsmanager.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return secretsmanager.NewFromConfig(cfg), nil
}

// Default returns a lazily initialized shared client for reuse across Lambda invocations.
func Default(ctx context.Context) (*secretsmanager.Client, error) {
	once.Do(func() {
		defaultClient, defaultErr = New(ctx)
	})

	return defaultClient, defaultErr
}

// GetSecretString fetches the raw SecretString value for the given secret ID.
func GetSecretString(ctx context.Context, secretID string) (string, error) {
	client, err := Default(ctx)
	if err != nil {
		return "", fmt.Errorf("create secrets manager client: %w", err)
	}

	return getSecretString(ctx, client, secretID)
}

// GetSecretValue fetches a JSON secret and returns a single value by key.
func GetSecretValue(ctx context.Context, secretID, key string) (string, error) {
	client, err := Default(ctx)
	if err != nil {
		return "", fmt.Errorf("create secrets manager client: %w", err)
	}

	return getSecretValue(ctx, client, secretID, key)
}

// UnmarshalSecret fetches a JSON secret and decodes it into the provided target struct.
func UnmarshalSecret(ctx context.Context, secretID string, target any) error {
	client, err := Default(ctx)
	if err != nil {
		return fmt.Errorf("create secrets manager client: %w", err)
	}

	return unmarshalSecret(ctx, client, secretID, target)
}

func getSecretString(ctx context.Context, client getSecretValueAPI, secretID string) (string, error) {
	output, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return "", fmt.Errorf("get secret value secret_id=%s: %w", secretID, err)
	}

	if output.SecretString == nil {
		return "", fmt.Errorf("secret string is empty for secret_id=%s", secretID)
	}

	return *output.SecretString, nil
}

func getSecretValue(ctx context.Context, client getSecretValueAPI, secretID, key string) (string, error) {
	secretString, err := getSecretString(ctx, client, secretID)
	if err != nil {
		return "", err
	}

	// This helper is intentionally narrow: it is meant for flat JSON secrets such as API keys.
	values := make(map[string]string)
	if err := json.Unmarshal([]byte(secretString), &values); err != nil {
		return "", fmt.Errorf("unmarshal secret json secret_id=%s: %w", secretID, err)
	}

	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("secret key=%s not found in secret_id=%s", key, secretID)
	}

	return value, nil
}

func unmarshalSecret(ctx context.Context, client getSecretValueAPI, secretID string, target any) error {
	secretString, err := getSecretString(ctx, client, secretID)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(secretString), target); err != nil {
		return fmt.Errorf("unmarshal secret json secret_id=%s: %w", secretID, err)
	}

	return nil
}
