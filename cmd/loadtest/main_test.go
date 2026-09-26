package main

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	t.Parallel()

	got := summarize([]time.Duration{4 * time.Millisecond, time.Millisecond, 3 * time.Millisecond, 2 * time.Millisecond})
	if got.Count != 4 || got.P50MS != 2 || got.P95MS != 4 || got.P99MS != 4 {
		t.Fatalf("unexpected percentiles: %+v", got)
	}
	if empty := summarize(nil); empty.Count != 0 {
		t.Fatalf("unexpected empty result: %+v", empty)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	valid := options{URL: "ws://localhost:8080/api/v1/opamp", Count: 1, Rate: 1, Duration: time.Second, ReportInterval: time.Second}
	if err := validate(valid); err != nil {
		t.Fatal(err)
	}
	valid.OpsInterval = time.Second
	if err := validate(valid); err == nil {
		t.Fatal("management operations must require credentials")
	}
}
