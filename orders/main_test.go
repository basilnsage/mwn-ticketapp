package main

import (
	"os"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func unsetEnvVars() error {
	for _, envVar := range []string{mongoConnStr, jwtSignKey, natsClusterId, natsClientId, natsConnStr} {
		if err := os.Unsetenv(envVar); err != nil {
			return err
		}
	}
	return nil
}

func TestValidateEnvVars(t *testing.T) {
	if err := unsetEnvVars(); err != nil {
		t.Fatal(err)
	}

	{
		got := validateEnvVars()
		sort.Strings(got)
		want := []string{
			"missing JWT HS256 signing key: JWT_SIGN_KEY",
			"missing NATS client ID: NATS_CLIENT_ID",
			"missing NATS cluster ID: NATS_CLUSTER_ID",
			"missing NATS connection string: NATS_CONN_STR",
			"missing mongo connection: MONGO_CONN_STR",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("diff: (-want +got)\n%v", diff)
		}
	}

	for _, envVar := range []string{mongoConnStr, jwtSignKey, natsClusterId, natsClientId, natsConnStr} {
		if err := os.Setenv(envVar, "test value"); err != nil {
			t.Fatal(err)
		}
	}
	{
		got := validateEnvVars()
		var want []string
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("diff: (-want +got)\n%v", diff)
		}
	}
}
