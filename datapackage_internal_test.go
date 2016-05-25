package datapackage

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"golang.org/x/crypto/openpgp/armor"
)

// TestEncrypt tests the datapackage.encrypt functionality.
func TestEncrypt(t *testing.T) {
	in := new(bytes.Buffer)

	encMsg, err := encrypt(in, strings.NewReader(keyRing))
	if err != nil {
		t.Fatalf("packer tests: error adding encryption: %s", err)
	}

	_, err = encMsg.Write([]byte(testMsg))
	if err != nil {
		t.Fatalf("packer tests: error writing message: %s", err)
	}
	encMsg.Close()

	decMsg, err := decrypt(in, strings.NewReader(keyRing), strings.NewReader(keyPass))
	if err != nil {
		t.Fatalf("packer test: error adding decryption: %s", err)
	}

	out, err := ioutil.ReadAll(decMsg)
	if err != nil {
		t.Fatalf("packer test: error decrypting message: %s", err)
	}

	if string(out) != testMsg {
		t.Fatalf("packer tests: round-trip message (%s) does not equal original (%s)", string(out), testMsg)
	}

}

const testMsg = `A test message`

// TestDecrypt tests the datapackage.decrypt functionality.
func TestDecrypt(t *testing.T) {
	// Un-armor encrypted message as that's what decrypt expects.
	encMsg, err := armor.Decode(strings.NewReader(msgEnc))
	if err != nil {
		t.Fatalf("packer tests: error ASCII decoding encrypted message: %s", err)
	}

	r, err := decrypt(encMsg.Body, strings.NewReader(keyRing), strings.NewReader(keyPass))
	if err != nil {
		t.Fatalf("packer tests: error adding decryption: %s", err)
	}

	newMsgDec, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("packer tests: error reading from DecryptingReader: %s", err)
	}

	if strings.TrimSpace(string(newMsgDec)) != msgDec {
		t.Fatalf("packer tests: decrypted message (%s) does not equal original (%s)", newMsgDec, msgDec)
	}
}

const msgDec = `Secret message`
const msgEnc = `-----BEGIN PGP MESSAGE-----

hQEMA3PhYU3mA1WKAQgAr2BSOwwKIlkA24Ieq+q58lo8bva7veFIXtaQ83qg9avK
kdrHU85oqv4Zb0U3BLNZYRIhxI+2t+N5SrW0ZrMB/DChfVoSQvcvlPFwpr03fNHN
nztnx1bFEkNpLr6ZAgQWMNFN4NlLuJrRoShp/9e0/aEd/Z0nLkF9uW/4Z4BxIwAr
W2pwfOS0ALk2DtCBwuI4rIg2w+Se47GIiWPX/+/EBzTH4hk+kvqf80PIUTpjsDM+
MGvq6MOcj5FDdcVUwPMHD5h5VP3JV90BS8aqfxOUSF0K1jt6LUxJkrv0LVHk6jR/
dVSEUGd2UyalqWzaRxzgVtvpBuiaOEcgmLtFIIF0y9JRAX6bBe6Rp9V58IMUjefc
juqwbEe/sLI3IVpCFOs4MxgY8BtuwD/MXIxsuzbrFb0kn0oGPhMBfxsPqTM+26hH
XlPN5TnsrhzA9OXZEnAES94k
=I3dd
-----END PGP MESSAGE-----`
const keyPass = `password`
const keyRing = `-----BEGIN PGP PRIVATE KEY BLOCK-----

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
=UQBW
-----END PGP PRIVATE KEY BLOCK-----
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBFaFbsEBCADJJfmo0QR075c/brBsKQBps0YYAQdvwRcFDRMGxbwBCvpHiOXy
0nmJ1qC3zD17VWYxuqRFo9mPlkGau4awEFlGV5UHp3iIedILjkiYkEWkciVuk/7T
bOwP2Ji4b3TNZ1LAqLon8+sYGIX7++iphORFzpZJMY2Y5pPg298n/U/Pp0CjIlwf
Xc7QKki2l3GLdgozXHgQAjoqscTz3pBNlMYBwbXz3ghAnqIUACdfUaM1WTXhgCRq
d/1FGPCMgvswPjrXUvdbbjw85VtYGHjLIGWymQqAJ7sSSwW84vSvwGFJQ5MPH9VR
TnDGfG2r+JKycrDW4cfufeaA8WLY6hIvkMddABEBAAG0NlRlc3RpbmcgS2V5IChE
TyBOT1QgVVNFISBURVNUSU5HIE9OTFkhKSA8dGVzdEBrZXkuY29tPokBNwQTAQoA
IQUCVoVuwQIbAwULCQgHAwUVCgkICwUWAgMBAAIeAQIXgAAKCRBwhpkMdWUeSChw
CACx2oPbFxLX3nqAfd5AK4it76xamLSiXZjkbSccc+7ZT/29OawCIJiYifKRNAlY
ghvjMqAsDmjuFzUoFGQSmM/3TnxxI866qP3BCLFvwyyKew20OVU7eo8NP1T7d+eI
x5hkUW7AoY7VPgpu/KVId8bNbGWYMZnwSAgV9abKcJwKV+knGrwshcKMEmOA2l1N
2ZCUAB9Q7Fyw3ql+NQh8zFQk9SmRAvl2JVXeuoCFEuYGBQCpVGE1kgUKNAgRfJIG
20a7qppcgrJqZiItjiDsvsy+X7TM1VgVJYtaxitTM8qqR3j1ntMyTCWk617eBSBc
qpABgP01apx5mSK8NndH2R8VuQENBFaFbsEBCADqP5uYYx5nxtDkhP849qSz0WjX
EW4FjHUjAC0lxyDWlwXd89G5EXspO/PtELDSbl16eFFwV0I1zwBxndTfSztgjwh7
TmF/F4bjBmAaH+LvksPYF8C84V3qCt3Yz/yUrhtesBS7OBeFLwDHZYQoj03am1Dl
aawFtYYWvEpua2FeW4wh1RazTleoSB95DGz10fbF9oR/TIQ3OBPTZBoV79EBcrL2
uz55gi467ebaNaXe2xenlH1IEiDm61lgrt3om3+YR00GKX4fzub8h7bcptH+mstF
vgih/xqTrwNLiyU+O9acsUuMvEPDxB1kDuSQcAvxHO/iI5jYtAeGnz+DwCklABEB
AAGJAR8EGAEKAAkFAlaFbsECGwwACgkQcIaZDHVlHkinLwgAxyEO+UAXMrGWCpMQ
zNXqbWkeXlHfQhqMEi4PdTRfs1VA+rju6D1xSOfZhJVqrsTXoJZJIm2Fxu1JY9yY
xBRhm8NGVH93VXQnBVe3WdYnBihKBWavqiYBIpbNvVwnKlL2p1IT5/zcK1nJSSjo
gPNSghOXOnVt1H0Ou7h1iwEXfhqAI7gq4Imt8yV5whOMcO/jVJyP4qR5HXxgFrKR
oV77WQ97d2SQ+jHxs1G6VhSu2u31ja0f+4C0BcVsulUQys3QmgEo10xv8TwN5+42
VshBa17tzRyuUghhz7p1X5rYgbYgqiAMxRPF45HrWmM8yBHt1En7cImNQvXpLEji
BBTn3w==
=435b
-----END PGP PUBLIC KEY BLOCK-----`
