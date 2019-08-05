package log2

import (
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestHclog2LoggerImplementsInterfaces(t *testing.T) {
	var logger interface{} = NewHclog2Logger(L())
	if _, ok := logger.(hclog.Logger); !ok {
		t.Fatalf("logger does not implement hclog.Logger")
	}
}
