package bahttext

import "testing"

func TestFromFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "ศูนย์บาทถ้วน"},
		{1, "หนึ่งบาทถ้วน"},
		{11, "สิบเอ็ดบาทถ้วน"},
		{21, "ยี่สิบเอ็ดบาทถ้วน"},
		{100, "หนึ่งร้อยบาทถ้วน"},
		{101, "หนึ่งร้อยเอ็ดบาทถ้วน"},
		{111, "หนึ่งร้อยสิบเอ็ดบาทถ้วน"},
		{1000, "หนึ่งพันบาทถ้วน"},
		{25750.50, "สองหมื่นห้าพันเจ็ดร้อยห้าสิบบาทห้าสิบสตางค์"},
		{1000000, "หนึ่งล้านบาทถ้วน"},
		{1000001, "หนึ่งล้านเอ็ดบาทถ้วน"},
		{2500000, "สองล้านห้าแสนบาทถ้วน"},
		{1000000000000, "หนึ่งล้านล้านบาทถ้วน"},
		{0.25, "ยี่สิบห้าสตางค์"},
		{0.005, "หนึ่งสตางค์"}, // ปัดขึ้น round-half-up
		{-150.75, "ลบหนึ่งร้อยห้าสิบบาทเจ็ดสิบห้าสตางค์"},
		{33712.94, "สามหมื่นสามพันเจ็ดร้อยสิบสองบาทเก้าสิบสี่สตางค์"},
	}
	for _, c := range cases {
		got, err := FromFloat(c.in)
		if err != nil {
			t.Fatalf("FromFloat(%v) error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("FromFloat(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFromFloatOutOfRange(t *testing.T) {
	for _, bad := range []float64{1e300} {
		if _, err := FromFloat(bad); err == nil {
			t.Errorf("FromFloat(%v) expected error", bad)
		}
	}
}
