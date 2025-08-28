package converter

import "testing"

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"asn", "Asn"},
		{"asnBe", "AsnBe"},
		{"ASN", "ASN"},
		{"ASNBe", "ASNBe"},
		{"asn_be", "AsnBe"},
		{"asn_be_foo", "AsnBeFoo"},
		{"foo", "Foo"},
		{"FOO", "FOO"},
		{"fooBar", "FooBar"},
		{"FOOBar", "FOOBar"},
		{"fooBAR", "FooBAR"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := snakeToCamel(tt.in); got != tt.out {
			t.Errorf("snakeToCamel(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
