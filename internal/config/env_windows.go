//go:build windows

package config

func mapEnvKey(key string) string {
	if key == "HOSTNAME" {
		return "COMPUTERNAME"
	}
	return key
}
