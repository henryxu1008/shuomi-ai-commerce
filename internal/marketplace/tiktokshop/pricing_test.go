package tiktokshop

import "testing"

func TestUSCrossBorderPriceUsesActualSelectedWarehouseType(t *testing.T) {
	direct := Warehouse{ID: "direct", Type: "SALES_WAREHOUSE", EffectStatus: "ENABLED", SubType: "CB_DIRECT_SHIPPING_WAREHOUSE"}
	local := direct
	local.ID = "us-local"
	local.SubType = "CB_OVERSEA_WAREHOUSE"
	for _, test := range []struct {
		price      string
		warehouses []Warehouse
		valid      bool
	}{
		{"800.00", []Warehouse{direct}, true}, {"800.01", []Warehouse{direct}, false},
		{"7600.00", []Warehouse{local}, true}, {"7600.01", []Warehouse{local}, false},
		{"801", []Warehouse{local, direct}, false}, {"0.01", []Warehouse{direct}, true},
		{"0", []Warehouse{direct}, false}, {"1e2", []Warehouse{direct}, false}, {"1.001", []Warehouse{direct}, false}, {"100", nil, false},
	} {
		if (ValidateUSSalePrice(test.price, test.warehouses) == nil) != test.valid {
			t.Errorf("price %s valid=%v", test.price, test.valid)
		}
	}
	for _, change := range []func(*Warehouse){func(v *Warehouse) { v.EffectStatus = "DISABLED" }, func(v *Warehouse) { v.Type = "RETURN_WAREHOUSE" }, func(v *Warehouse) { v.SubType = "UNKNOWN" }} {
		v := local
		change(&v)
		if ValidateUSSalePrice("10", []Warehouse{v}) == nil {
			t.Fatal("unverified warehouse accepted")
		}
	}
}
