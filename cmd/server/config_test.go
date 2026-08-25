package main

import "testing"

func TestAddressFromPortUsesLoopback(t *testing.T) {
	if got := addressFromPort("19444"); got != "127.0.0.1:19444" {
		t.Fatalf("got %q", got)
	}
	if got := addressFromPort("invalid"); got != "" {
		t.Fatalf("非法 PORT 应忽略，got %q", got)
	}
}

func TestParseConfigDefaultsToHighLoopbackPort(t *testing.T) {
	t.Setenv("PORT", "")
	configuration, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.address != defaultAddress {
		t.Fatalf("address=%s, want %s", configuration.address, defaultAddress)
	}
}

func TestParseConfigHonorsExplicitAddress(t *testing.T) {
	t.Setenv("PORT", "19999")
	configuration, err := parseConfig([]string{"-addr=127.0.0.1:19555"})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.address != "127.0.0.1:19555" {
		t.Fatalf("address=%s", configuration.address)
	}
}
