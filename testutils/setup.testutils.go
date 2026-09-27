package testutils

import (
	"context"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"os"
	"testing"

	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func SetupContractEnv() {
	os.Setenv("ENV", "TESTNET")
	os.Setenv("RPC_URL", "https://kartel-testnet.alt.technology")
	os.Setenv("KEEPER", "a234c23b58dee670d6be91736cada2b6005a56d86819ff83a479a0512dae1a23")
}

func SetMainnetEnv() {
	os.Setenv("ENV", "MAINNET")
	contractUtils.Init()
}

func SetTestnetEnv() {
	os.Setenv("ENV", "TESTNET")
	contractUtils.Init()
}

func SetupBalanceEnv() {}

func SetupBalanceMainnetEnv() {
	SetMainnetEnv()
	os.Setenv("PORT", "8092")
	os.Setenv("KEEPER", "a234c23b58dee670d6be91736cada2b6005a56d86819ff83a479a0512dae1a23")
	os.Setenv("DISCORD_NETWORK_SYNC_WEBHOOK", "test")
	os.Setenv("ORACLE_SERVER_URL", "test")
}

func ResetEnv() {
	os.Clearenv()
}

func SetupEngineEnv() {
	os.Setenv("IS_DEV", "1")
	os.Setenv("BALANCE_SERVER_URL", "http://localhost:8092")
	os.Setenv("ORACLE_SERVER_URL", "https://oracle.hundred.exchange")
	os.Setenv("API_SERVER_URL", "http://localhost:8090")
	os.Setenv("ENGINE_URL", "http://localhost:8091")
	os.Setenv("PORT", "8091")
}

func SetupDBEnv(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	container, err := pgcontainer.Run(ctx,
		"postgres:16-alpine",
		pgcontainer.WithDatabase("testdb"),
		pgcontainer.WithUsername("test"),
		pgcontainer.WithPassword("test"),
		pgcontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	os.Setenv("DSN", dsn)
}
