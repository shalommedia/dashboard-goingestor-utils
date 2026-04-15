package secretsmanagerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
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
	secretRef := redactedSecretRef(secretID)

	output, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return "", fmt.Errorf("get secret value secret_ref=%s: %w", secretRef, err)
	}

	if output.SecretString == nil {
		return "", fmt.Errorf("secret string is empty secret_ref=%s", secretRef)
	}

	return *output.SecretString, nil
}

func getSecretValue(ctx context.Context, client getSecretValueAPI, secretID, key string) (string, error) {
	secretRef := redactedSecretRef(secretID)

	secretString, err := getSecretString(ctx, client, secretID)
	if err != nil {
		return "", fmt.Errorf("read secret payload secret_ref=%s: %w", secretRef, err)
	}

	// This helper is intentionally narrow: it is meant for flat JSON secrets such as API keys.
	values := make(map[string]string)
	if err := json.Unmarshal([]byte(secretString), &values); err != nil {
		return "", fmt.Errorf("unmarshal secret json: %w", err)
	}

	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("requested secret key not found secret_ref=%s", secretRef)
	}

	return value, nil
}

func unmarshalSecret(ctx context.Context, client getSecretValueAPI, secretID string, target any) error {
	secretRef := redactedSecretRef(secretID)

	secretString, err := getSecretString(ctx, client, secretID)
	if err != nil {
		return fmt.Errorf("read secret payload secret_ref=%s: %w", secretRef, err)
	}

	if err := json.Unmarshal([]byte(secretString), target); err != nil {
		return fmt.Errorf("unmarshal secret json secret_ref=%s: %w", secretRef, err)
	}

	return nil
}

func redactedSecretRef(secretID string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(secretID))

	return fmt.Sprintf("fnv64a:%016x", hasher.Sum64())
}
