// +build darwin

package originaldest

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// cSpell: ignore CFLAGS DIOCNATLOOK
// cSpell: ignore pfvar pfioc
// cSpell: ignore        sxport  saddr
// cSpell: ignore rport rdxport rdaddr
// cSpell: ignore rport  dxport  daddr

// #cgo CFLAGS: -I./darwin/include
// #include <sys/ioctl.h>
// #define PRIVATE
// #include <net/pfvar.h>
// #undef PRIVATE
import "C"

var devicePath = "/dev/pf"

func ioctl(fd uintptr, cmd uintptr, ptr unsafe.Pointer) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, uintptr(ptr))
	if err != 0 {
		return err
	}
	return nil
}

// GetOriginalDst gets IP-Address and Port to which the client likes to connect
func GetOriginalDst(clientConn *net.TCPConn) (string, uint16, *net.TCPConn, error) {
	client := clientConn.RemoteAddr()
	local := clientConn.LocalAddr()

	clientAddr, _ := client.(*net.TCPAddr)
	localAddr, _ := local.(*net.TCPAddr)
	f, err := os.OpenFile(devicePath, os.O_RDWR, 0644)
	if err != nil {
		return "", 0, clientConn, err
	}
	fd := f.Fd()
	pnl := new(C.struct_pfioc_natlook)
	pnl.direction = C.PF_OUT
	pnl.af = C.AF_INET
	pnl.proto = C.IPPROTO_TCP
	copy(pnl.saddr.pfa[0:4], clientAddr.IP)
	clientPort := make([]byte, 2)
	binary.BigEndian.PutUint16(clientPort, uint16(clientAddr.Port))
	copy(pnl.sxport[:], clientPort)

	localPort := make([]byte, 2)
	copy(pnl.daddr.pfa[0:4], localAddr.IP)
	binary.BigEndian.PutUint16(localPort, uint16(localAddr.Port))
	copy(pnl.dxport[:], localPort)

	// Do lookup to fullfill pnl.rdxport and pnl.rdaddr fields
	if err := ioctl(fd, C.DIOCNATLOOK, unsafe.Pointer(pnl)); err != nil {
		return "", 0, clientConn, err
	}

	rport := make([]byte, 2)
	copy(rport, pnl.rdxport[:2])
	fmt.Println("port", binary.BigEndian.Uint16(rport))
	fmt.Println("addr", pnl.rdaddr.pfa[:4])

	return "", 0, clientConn, nil
}
