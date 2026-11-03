package opnsensetest

import "fmt"

// registerAssignments makes interface assignments behave like OPNsense: items are keyed by
// identifier, new ones get the next free optN, and lan and wan exist from the start.
//
// OPNsense keeps dhcp6_request_dns inverted in config.xml as dhcp6_norequest_dns, and saves it
// only when the IPv6 type is dhcp6 (NetworkInterfaceField::toLegacy). For any other type the
// flag is dropped, and reads back as true.
func registerAssignments(s *Server) {
	c := s.collection("interfaces/assignment", "interface")
	c.OnWrite = func(_ *Server, uuid string, posted Record) {
		type6, ok := posted["type6"]
		if !ok {
			type6 = c.items[uuid]["type6"]
		}
		if type6 != "dhcp6" {
			posted["dhcp6_request_dns"] = "1"
		}
	}
	c.NewID = func(_ *Server, rec Record) string {
		for n := 1; ; n++ {
			id := fmt.Sprintf("opt%d", n)
			if _, used := c.items[id]; !used {
				rec["identifier"] = id
				return id
			}
		}
	}
	for id, dev := range map[string]string{"lan": "vtnet0", "wan": "vtnet1"} {
		rec := c.Defaults()
		rec["identifier"], rec["if"], rec["descr"], rec["enable"] = id, dev, map[string]string{"lan": "LAN", "wan": "WAN"}[id], "1"
		rec["dhcp6_request_dns"] = "1"
		c.items[id] = rec
	}
}
