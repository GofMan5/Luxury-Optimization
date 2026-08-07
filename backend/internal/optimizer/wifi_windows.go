package optimizer

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var wifiCapability struct {
	sync.Once
	available bool
}

func hasPhysicalWiFi() bool {
	wifiCapability.Do(func() { wifiCapability.available = detectPhysicalWiFi() })
	return wifiCapability.available
}

func detectPhysicalWiFi() bool {
	size := uint32(16 << 10)
	const maximum = uint32(1 << 20)
	flags := uint32(windows.GAA_FLAG_SKIP_UNICAST | windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_DNS_SERVER | windows.GAA_FLAG_INCLUDE_ALL_INTERFACES)
	for range 3 {
		if size == 0 || size > maximum {
			return false
		}
		buffer := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0])) // #nosec G103 -- bounded IP Helper output buffer retained for the full traversal.
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, first, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return false
		}
		base := uintptr(unsafe.Pointer(&buffer[0]))
		limit := base + uintptr(len(buffer))
		for current, count := first, 0; current != nil && count < 512; current, count = current.Next, count+1 {
			address := uintptr(unsafe.Pointer(current))
			if address < base || address > limit-unsafe.Sizeof(*current) || current.Length < uint32(unsafe.Sizeof(*current)) {
				return false
			}
			if current.IfType == windows.IF_TYPE_IEEE80211 && current.PhysicalAddressLength > 0 && current.PhysicalAddressLength <= uint32(len(current.PhysicalAddress)) {
				return true
			}
		}
		return false
	}
	return false
}
