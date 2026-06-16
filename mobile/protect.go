package mobile

import (
	"fmt"
	"log"
)

type ProtectFunc interface {
	ProtectFd(fd int64) bool
}

var (
	protectCallback ProtectFunc
	protectEnabled  bool
)

func SetProtectFunc(p ProtectFunc) {
	protectCallback = p
	protectEnabled = (p != nil)
	
	if protectEnabled {
		log.Println("[protect] ✅ Callback registered")
	} else {
		log.Println("[protect] ⚠️ Callback cleared")
	}
}

func protectSocket(fd int) error {
	if !protectEnabled || protectCallback == nil {
		log.Printf("[protect] ⚠️ Skipping fd=%d (callback not set)", fd)
		return nil
	}
	
	success := protectCallback.ProtectFd(int64(fd))
	if !success {
		return fmt.Errorf("protect failed for fd=%d", fd)
	}
	
	log.Printf("[protect] ✅ Socket protected: fd=%d", fd)
	return nil
}

func GetProtectEnabled() bool {
	return protectEnabled
}
