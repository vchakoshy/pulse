package mtproto

import "errors"

const (
	appId   = 75354
	appHash = "af224f7c58a13fdda0db382ce9ab7719"
	// appId   = 22457
	// appId   = 149095
	// appHash = "9a6806c43ac14163cc9adb2864797f10"
	// serverAddr = "91.108.4.182:443"
	// serverAddr = "149.154.167.91:443"
	// serverAddr = "149.154.167.40:443"
	// serverAddr = "149.154.167.50:443"
	serverAddr = "127.0.0.1:8888"
	// serverAddr   = "149.154.175.50:443 "
	readDeadLine = 120
)

var (
	ErrTelegramTimeOut = errors.New("telegram time out.")
	ErrUsernameInvalid = errors.New("USERNAME_INVALID")
)
