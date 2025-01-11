package feed_api

import "fmt"

type RemoteIDE interface {
	fmt.Stringer

	Name() string
	Build() string
	PackageType() string
}
