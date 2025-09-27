package authjwt

import "testing"

func TestStuv(t *testing.T) {
	if Stub() != "ok" {
		t.Fatalf("want ok, but got %s", Stub())
	}
}
