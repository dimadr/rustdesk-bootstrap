package main

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	serverValue = "example.com"
	keyValue = "public-key-for-tests"
	desiredConfig["custom-rendezvous-server"] = serverValue
	desiredConfig["relay-server"] = serverValue
	desiredConfig["key"] = keyValue
	os.Exit(m.Run())
}

func TestRewriteTomlUpdatesOnlyTargetKeys(t *testing.T) {
	input := strings.Join([]string{
		"access-mode = \"full\"",
		"custom-rendezvous-server = \"old.example\"",
		"relay-server = \"old-relay.example\"",
		"key = \"old-key\"",
		"direct-server = \"unchanged\"",
		"",
	}, "\r\n")

	got, changed := rewriteToml([]byte(input))
	if !changed {
		t.Fatal("expected config to change")
	}

	want := strings.Join([]string{
		"access-mode = \"full\"",
		"custom-rendezvous-server = \"example.com\"",
		"relay-server = \"example.com\"",
		"key = \"public-key-for-tests\"",
		"direct-server = \"unchanged\"",
		"",
	}, "\r\n")

	if string(got) != want {
		t.Fatalf("unexpected rewrite:\n%s", string(got))
	}
}

func TestRewriteTomlAddsMissingKeys(t *testing.T) {
	input := []byte("access-mode = \"full\"\n")

	got, changed := rewriteToml(input)
	if !changed {
		t.Fatal("expected config to change")
	}

	text := string(got)
	for _, line := range []string{
		"access-mode = \"full\"",
		"custom-rendezvous-server = \"example.com\"",
		"relay-server = \"example.com\"",
		"key = \"public-key-for-tests\"",
	} {
		if !strings.Contains(text, line) {
			t.Fatalf("missing line %q in:\n%s", line, text)
		}
	}
}
