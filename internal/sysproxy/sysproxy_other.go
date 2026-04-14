//go:build !darwin && !linux

package sysproxy

func newPlatformProxy() SystemProxy {
	return nil
}
