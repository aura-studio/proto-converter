package converter

import "testing"

func TestToCamel(t *testing.T) {
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
		if got := toCamel(tt.in); got != tt.out {
			t.Errorf("snakeToCamel(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}

func TestToSnake(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"FooBar", "foo_bar"},
		{"fooBarBaz", "foo_bar_baz"},
		{"FOOBar", "f_o_o_bar"},
		{"fooBAR", "foo_b_a_r"},
		{"foo_bar", "foo_bar"},
		{"foo-bar", "foo_bar"},
		{"foo.bar", "foo_bar"},
		{"foo bar", "foo_bar"},
		{"foo__bar", "foo_bar"},
		{"fooBar1", "foo_bar1"},
		{"foo1Bar", "foo1_bar"},
		{"foo1bar2", "foo1bar2"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := toSnake(tt.in); got != tt.out {
			t.Errorf("toSnake(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
