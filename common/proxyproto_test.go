package common

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

func TestProxyV2RoundTrip(t *testing.T) {
	for _, tc := range []struct {
		stream bool
		src    string
		dst    string
	}{{true, "198.51.100.7", "145.241.199.19"}, {false, "2001:db8::1", "2001:db8::2"}} {
		b := append(EncodeProxyV2(tc.stream, net.ParseIP(tc.src), 41234, net.ParseIP(tc.dst), 27900), "payload"...)
		h, rest, err := ParseProxyV2(b)
		if err != nil || !h.SrcIP.Equal(net.ParseIP(tc.src)) || h.SrcPort != 41234 || !h.DstIP.Equal(net.ParseIP(tc.dst)) || h.DstPort != 27900 || string(rest) != "payload" {
			t.Fatalf("%+v %q %v", h, rest, err)
		}
	}
	if _, _, err := ParseProxyV2([]byte("\\lc\\1\\challenge")); err != ErrNotProxy {
		t.Fatal("a GameSpy packet parsed as a header")
	}
}

func TestProxyListenerReportsTheClient(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pl := ProxyListener{l}
	defer pl.Close()
	go func() {
		c, _ := net.Dial("tcp", l.Addr().String())
		c.Write(EncodeProxyV2(true, net.ParseIP("203.0.113.9"), 5555, net.ParseIP("145.241.199.19"), 29900))
		c.Write([]byte("hi"))
		time.Sleep(200 * time.Millisecond)
		c.Close()
	}()
	// A connection without a header is dropped, not returned.
	go func() {
		c, _ := net.Dial("tcp", l.Addr().String())
		c.Write([]byte("no header here...."))
		time.Sleep(6 * time.Second)
		c.Close()
	}()
	c, err := pl.Accept()
	if err != nil {
		t.Fatal(err)
	}
	if c.RemoteAddr().String() != "203.0.113.9:5555" {
		t.Fatalf("remote %v", c.RemoteAddr())
	}
	b := make([]byte, 2)
	if _, err := io.ReadFull(bufio.NewReader(c), b); err != nil || string(b) != "hi" {
		t.Fatalf("data %q %v", b, err)
	}
}

// fakeRelay plays wfc-front: it frames a player's datagram to the service's proxy socket and
// unframes what comes back.
func TestDualPacketConn(t *testing.T) {
	pc, err := ListenGameUDP("127.0.0.1:0", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	d := pc.(*DualPacketConn)
	relay, _ := net.ListenPacket("udp", "127.0.0.1:0")
	defer relay.Close()
	direct, _ := net.ListenPacket("udp", "127.0.0.1:0")
	defer direct.Close()

	player := &net.UDPAddr{IP: net.ParseIP("198.51.100.7"), Port: 40000}
	relay.WriteTo(append(EncodeProxyV2(false, player.IP, player.Port, net.ParseIP("145.241.199.19"), 27900), "\x09hello"...), d.proxy.LocalAddr())
	direct.WriteTo([]byte("\x09plain"), d.direct.LocalAddr())

	got := map[string]string{}
	buf := make([]byte, 100)
	for i := 0; i < 2; i++ {
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			t.Fatal(err)
		}
		got[from.String()] = string(buf[:n])
	}
	if got[player.String()] != "\x09hello" || got[direct.LocalAddr().String()] != "\x09plain" {
		t.Fatalf("read %q", got)
	}

	// A reply (or an unsolicited packet) to the relayed player goes back framed through the relay.
	if _, err := pc.WriteTo([]byte("reply"), player); err != nil {
		t.Fatal(err)
	}
	relay.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := relay.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	h, rest, err := ParseProxyV2(buf[:n])
	if err != nil || !h.DstIP.Equal(player.IP) || h.DstPort != player.Port || h.SrcPort != 27900 || string(rest) != "reply" {
		t.Fatalf("framed reply %+v %q %v", h, rest, err)
	}
	// A direct player is answered directly.
	if _, err := pc.WriteTo([]byte("direct"), direct.LocalAddr()); err != nil {
		t.Fatal(err)
	}
	direct.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err = direct.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "direct" {
		t.Fatalf("direct reply %q %v", buf[:n], err)
	}
}

func TestListenGameUDPPlainWithoutProxy(t *testing.T) {
	pc, err := ListenGameUDP("127.0.0.1:0", "")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	if _, ok := pc.(*DualPacketConn); ok {
		t.Fatal("no proxy address must give a plain socket")
	}
}
