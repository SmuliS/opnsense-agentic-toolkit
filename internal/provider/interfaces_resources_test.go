package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInterfaceAssignment(t *testing.T) {
	h := newHarness(t, nil, NewNetworkInterfaceResource())
	var got genNetworkInterfaceModel
	if _, ok := h.create(genNetworkInterfacePlan(), &got); !ok {
		t.Fatal("create failed")
	}
	if got.ID.ValueString() != "opt1" || got.Identifier.ValueString() != "opt1" {
		t.Fatalf("id %v, identifier %v; want opt1", got.ID, got.Identifier)
	}
	if h.fake.Applies["interfaces/assignment/reconfigure"] != 1 {
		t.Fatalf("applies %v", h.fake.Applies)
	}
	for in, want := range map[string]string{"lan": "lan", "WAN": "wan", "IOT": "opt1"} {
		if id, ok := h.importID(in); !ok || id != want {
			t.Errorf("import %q = %q, want %q", in, id, want)
		}
	}
	if _, ok := h.importID("nope"); ok {
		t.Error("import of an unknown interface succeeded")
	}
}

// An optional masked or network field accepts an explicit "", because OPNsense checks only a
// non-empty value. `tofu plan -generate-config-out` writes "" for every empty field of an
// imported interface, such as the IPv4 address of a DHCP WAN, and `= ""` clears a field.
func TestInterfaceEmptyMaskedFields(t *testing.T) {
	h := newHarness(t, nil, NewNetworkInterfaceResource())
	plan := genNetworkInterfacePlan()
	plan.Dhcp6Ifid = types.StringValue("0x1a")
	if d := h.validateStrings(plan); d.HasError() {
		t.Fatalf("valid plan rejected: %v", d)
	}
	var got genNetworkInterfaceModel
	st, ok := h.create(plan, &got)
	if !ok {
		t.Fatal("create failed")
	}
	id := got.ID.ValueString()

	bad := plan
	bad.Dhcp6Ifid = types.StringValue("1a")
	if !h.validateStrings(bad).HasError() {
		t.Error("dhcp6_ifid without 0x accepted")
	}
	bad = plan
	bad.IPv4Address = types.StringValue("10.30.0.1")
	if !h.validateStrings(bad).HasError() {
		t.Error("ipv4_address without a prefix accepted")
	}

	plan.ID, plan.Identifier = got.ID, got.Identifier
	for _, s := range []*types.String{&plan.Dhcp6Ifid, &plan.Dhcp6PrefixID, &plan.Track6Ifid, &plan.Track6PrefixID,
		&plan.IPv4Type, &plan.IPv4Address} {
		*s = types.StringValue("")
	}
	if d := h.validateStrings(plan); d.HasError() {
		t.Fatalf("explicit empty values rejected: %v", d)
	}
	h.update(st, plan, &got)
	rec, _ := h.fake.Get("interface", id)
	for _, f := range []string{"dhcp6_ifid", "ipaddr"} {
		if rec[f] != "" {
			t.Errorf("%s not cleared: %q", f, rec[f])
		}
	}

	// Import: the generated config is the state read after import, with every value explicit.
	if imp, ok := h.importID(id); !ok || imp != id {
		t.Fatalf("import = %q", imp)
	}
	imported := h.state()
	if d := imported.SetAttribute(h.ctx, pathID, id); d.HasError() {
		t.Fatal(d)
	}
	var config genNetworkInterfaceModel
	if _, ok := h.read(imported, &config); !ok {
		t.Fatal("read after import removed the resource")
	}
	for name, v := range map[string]types.String{"dhcp6_ifid": config.Dhcp6Ifid, "ipv4_address": config.IPv4Address} {
		if v.IsNull() || v.ValueString() != "" {
			t.Fatalf("imported %s = %v, want \"\"", name, v)
		}
	}
	if d := h.validateStrings(config); d.HasError() {
		t.Errorf("config generated on import rejected: %v", d)
	}
}
