// Package provisioning installs Apple provisioning profiles into the
// directory Xcode discovers them from.
package provisioning

type Installed struct {
	Name     string
	UUID     string
	DestPath string
}
