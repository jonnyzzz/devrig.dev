package unpack_api

import (
	"fmt"

	"jonnyzzz.com/devrig.dev/feed_api"
)

type UnpackedDownloadedRemoteIde interface {
	fmt.Stringer

	UnpackedHome() string
	RemoteIde() feed_api.RemoteIDE
}
