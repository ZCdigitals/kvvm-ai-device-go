package wake_on_lan

import "testing"

func TestMagicPacket(t *testing.T) {
	t.Run("create magic packet", func(t *testing.T) {
		expected := magicPacket{
			header: [6]byte{
				0xff, // 0
				0xff, // 1
				0xff, // 2
				0xff, // 3
				0xff, // 4
				0xff, // 5
			},
			payload: [16][6]byte{
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 0
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 1
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 2
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 3
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 4
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 5
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 6
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 7
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 8
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // 9
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // a
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // b
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // c
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // d
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // e
				[6]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab}, // f
			},
		}
		eb := expected.bytes()

		tt := func(mp *magicPacket, err error) {
			if err != nil {
				t.Errorf("create error %v", err)
				return
			}

			mb := mp.bytes()
			if len(mb) != len(eb) {
				t.Errorf("bytes length not same %d %d", len(mb), len(eb))
				return
			}

			for i := range len(mb) {
				if mb[i] != eb[i] {
					t.Errorf("byte at %d not same %d %d", i, mb[i], eb[i])
					return
				}
			}
		}

		mac1 := "01:23:45:67:89:AB"
		mp, err := newMagicPacket(mac1)
		tt(mp, err)

		mac2 := "01-23-45-67-89-AB"
		mp, err = newMagicPacket(mac2)
		tt(mp, err)

		mac3 := "0123.4567.89AB"
		mp, err = newMagicPacket(mac3)
		tt(mp, err)

		mac4 := "0123456789AB"
		mp, err = newMagicPacket(mac4)
		tt(mp, err)
	})
}

func TestUseIPs(t *testing.T) {
	t.Run("use ips", func(t *testing.T) {
		ips, err := useIPs()
		if err != nil {
			t.Errorf("error %v", err)
			return
		}

		for _, ip := range ips {
			t.Logf("ip %s", ip.String())
		}
	})
}

func TestSendWOL(t *testing.T) {
	t.Run("send wol", func(t *testing.T) {
		mac := "C4:75:AB:1A:07:1B"

		err := SendWOL(mac)
		if err != nil {
			t.Errorf("error %v", err)
			return
		}
	})
}
