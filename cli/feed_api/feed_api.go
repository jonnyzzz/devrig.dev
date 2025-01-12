package feed_api

import "fmt"

type RemoteIDE interface {
	fmt.Stringer

	Name() string
	Build() string
	PackageType() string

	// IdeType returns `intellij` for IntelliJ ides
	IdeType() string
}

type DownloadedRemoteIde interface {
	fmt.Stringer

	TargetFile() string
	RemoteIde() RemoteIDE
}
