package http2_test

import (
	"fmt"
	"testing"

	"github.com/sapnewala/life-agent/pkg/http2"
)

func TestJQRun(t *testing.T) {
	var doc string
	var err error
	doc, err = http2.JQRun(`{"a":"2,300억"}`, ".a")
	fmt.Println("doc=", doc)
	doc, err = http2.JQRun(`"2,300억"`, "ko2num", http2.DefaultJQFunctions)
	if err != nil {
		t.Fatalf("Failed to JQRun: %v", err)
	}
	fmt.Println("doc=", doc)
}
