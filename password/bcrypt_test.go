package password

import "testing"

func TestBcrypt(t *testing.T) {
	in := "123"
	out, err := BcryptHash(in)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("hashed password: ", out)

	if !BycyptCompare(in, out) {
		t.Errorf("Compare fail")
	}

}
