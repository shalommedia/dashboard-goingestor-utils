package secretsmanagerclient

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type mockSecretsClient struct {
	output *secretsmanager.GetSecretValueOutput
	err    error
}

func (m *mockSecretsClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.output, nil
}

func TestGetSecretString_WrapsWithoutLeakingSecretID(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("access denied")
	_, err := getSecretString(context.Background(), &mockSecretsClient{err: sentinelErr}, "prod/payment/token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected wrapped sentinel error, got: %v", err)
	}

	if strings.Contains(err.Error(), "prod/payment/token") {
		t.Fatalf("error leaked secret id: %v", err)
	}
}

func TestGetSecretValue_MissingKey_DoesNotLeakKeyOrSecretID(t *testing.T) {
	t.Parallel()

	secretPayload := `{"api_key":"value"}`
	_, err := getSecretValue(context.Background(), &mockSecretsClient{
		output: &secretsmanager.GetSecretValueOutput{SecretString: &secretPayload},
	}, "prod/payment/token", "missing_key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if strings.Contains(err.Error(), "prod/payment/token") {
		t.Fatalf("error leaked secret id: %v", err)
	}

	if strings.Contains(err.Error(), "missing_key") {
		t.Fatalf("error leaked requested key: %v", err)
	}
}

func TestUnmarshalSecret_InvalidJSON_DoesNotLeakSecretID(t *testing.T) {
	t.Parallel()

	invalid := `{"key":`
	err := unmarshalSecret(context.Background(), &mockSecretsClient{
		output: &secretsmanager.GetSecretValueOutput{SecretString: &invalid},
	}, "prod/payment/token", &struct{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if strings.Contains(err.Error(), "prod/payment/token") {
		t.Fatalf("error leaked secret id: %v", err)
	}
}
