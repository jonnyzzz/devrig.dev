package unpack_api

import (
	"cli/feed_api"
	"fmt"
)

type UnpackedDownloadedRemoteIde interface {
	fmt.Stringer

	UnpackedHome() string
	RemoteIde() feed_api.RemoteIDE
}
