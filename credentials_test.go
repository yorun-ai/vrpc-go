package vrpc_test

import (
	"go.yorun.ai/vrpc"
	"testing"
)

func TestCredentials(t *testing.T) {
	value, err := vrpc.EncodeCredentials(map[string]string{"token": "abc==", "key": "hello"})
	if err != nil || value != "key hello, token abc==" {
		t.Fatalf("%q %v", value, err)
	}
	for _, credentials := range []map[string]string{{"key": ""}, {"key": "a,b"}, {"key": "a\r\nb"}, {"bad name": "x"}, {"Key": "a", "key": "b"}, {"key": " padded "}} {
		if _, err := vrpc.EncodeCredentials(credentials); err == nil {
			t.Errorf("accepted invalid credentials")
		}
	}
}
