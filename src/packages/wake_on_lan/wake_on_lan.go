package wake_on_lan

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
)

type magicPacket struct {
	header  [6]byte
	payload [16][6]byte
}

// create magic packet
func newMagicPacket(mac string) (*magicPacket, error) {
	pm, err := pureMacAddress(mac)
	if err != nil {
		return nil, err
	}

	mb, err := hex.DecodeString(pm)
	if err != nil {
		return nil, err
	} else if len(mb) != 6 {
		return nil, fmt.Errorf("Invalid mac address %s", mac)
	}

	mp := magicPacket{
		// header is 0xffffffffffff
		header:  [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		payload: [16][6]byte{},
	}

	// repeat mac address 16 times
	for i := range 16 {
		copy(mp.payload[i][:], mb)
	}

	return &mp, nil
}

func (mp *magicPacket) bytes() []byte {
	b := make([]byte, 6+16*6)

	copy(b[:6], mp.header[:])

	offset := 6
	for i := 0; i < 16; i++ {
		copy(b[offset:offset+6], mp.payload[i][:])
		offset += 6
	}

	return b
}

// pure mac address, replace all `:` `-` `.`
func pureMacAddress(mac string) (string, error) {
	nm := strings.ReplaceAll(mac, ":", "")
	nm = strings.ReplaceAll(nm, "-", "")
	nm = strings.ReplaceAll(nm, ".", "")

	if len(nm) != 12 {
		return "", fmt.Errorf("Invalid mac address %s", mac)
	}

	return nm, nil
}

// use all avalible ips
func useIPs() ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ips := []net.IP{}

	for _, ifc := range interfaces {
		if (ifc.Flags & net.FlagLoopback) != 0 {
			// skip loop
			continue
		} else if (ifc.Flags & net.FlagUp) == 0 {
			// skip inactive
			continue
		}

		addrs, err := ifc.Addrs()
		if err != nil {
			log.Println("wake on lan read addresses warning", err)
			continue
		}

		for _, addr := range addrs {
			ipn, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := useIp(ipn)
			if ip == nil {
				continue
			}
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// use masked ipv4 address
func useIp(ipn *net.IPNet) net.IP {
	ip := ipn.IP.To4()
	if ip == nil {
		// not ipv4
		return nil
	}

	mask := ipn.Mask
	if len(mask) != net.IPv4len {
		// not ipv4
		return nil
	}

	nip := make(net.IP, net.IPv4len)
	for i := range net.IPv4len {
		// use mask
		nip[i] = ip[i] | ^mask[i]
	}

	return nip
}

const wakeOnLanPort = 9

func sendWOL(mac string, ip net.IP) error {
	addr := net.UDPAddr{
		IP:   ip,
		Port: wakeOnLanPort,
	}

	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// set broadcast
	conn.SetWriteBuffer(1024)

	mp, err := newMagicPacket(mac)
	if err != nil {
		return err
	}

	_, err = conn.Write(mp.bytes())
	if err != nil {
		return err
	}

	return nil
}

func SendWOL(mac string) error {
	ips, err := useIPs()
	if err != nil {
		return err
	}

	for _, ip := range ips {
		err := sendWOL(mac, ip)
		if err != nil {
			log.Println("wake on lan warning, send error", err)
		}
	}

	return nil
}
