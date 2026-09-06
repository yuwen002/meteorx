package signedurl

import (
	"net/url"
	"testing"
	"time"
)

const testSecret = "test-secret-abcdefghijklmnopqrstuvwxyz-1234567890"

func TestSignVerifyValid(t *testing.T) {
	expires := Expires(DefaultTTL)
	path := "01JZYZabcdefghijklmnopqrst.pdf"
	sig := Sign(path, testSecret, expires)
	if !Verify(path, testSecret, expires, sig) {
		t.Fatal("valid signature should pass verification")
	}
}

func TestSignVerifyRejectsTampering(t *testing.T) {
	expires := Expires(DefaultTTL)
	path := "01JZYZabcdefghijklmnopqrst.pdf"
	sig := Sign(path, testSecret, expires)

	cases := []struct {
		name    string
		path    string
		secret  string
		expires int64
		sig     string
	}{
		{"wrong path", "other.pdf", testSecret, expires, sig},
		{"wrong secret", path, "another-secret", expires, sig},
		{"tampered expires", path, testSecret, expires + 1, sig},
		{"tampered sig", path, testSecret, expires, "0000000000000000000000000000000000000000000000000000000000000000"},
		{"empty sig", path, testSecret, expires, ""},
		{"zero expires", path, testSecret, 0, sig},
	}
	for _, c := range cases {
		if Verify(c.path, c.secret, c.expires, c.sig) {
			t.Errorf("%s: verification should fail", c.name)
		}
	}
}

func TestBuild(t *testing.T) {
	path := "demo.png"
	expires := time.Now().Add(time.Minute).Unix()
	raw := Build("http://localhost:8081/uploads/", path, testSecret, expires)

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("Build returned invalid URL: %v", err)
	}
	if u.Path != "/uploads/demo.png" {
		t.Errorf("unexpected path: %s", u.Path)
	}
	q := u.Query()
	if q.Get(QueryExpires) == "" || q.Get(QuerySig) == "" {
		t.Fatalf("missing signature query params: %s", raw)
	}
	if !Verify(path, testSecret, expires, q.Get(QuerySig)) {
		t.Fatal("query signature in Build URL should verify")
	}
}
