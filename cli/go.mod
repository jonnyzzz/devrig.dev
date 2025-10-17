module jonnyzzz.com/devrig.dev

go 1.25

require (
	github.com/spf13/cobra v1.10.1
	github.com/ulikunitz/xz v0.5.15
	go.mozilla.org/pkcs7 v0.9.0
	gopkg.in/yaml.v3 v3.0.1
	jonnyzzz.com/devrig.dev/bootstrap v0.0.0
)

require golang.org/x/crypto v0.43.0

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/sys v0.37.0 // indirect
)

replace jonnyzzz.com/devrig.dev/bootstrap => ./bootstrap
