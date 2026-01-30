package playwright

import "embed"

//go:embed scripts/launch-server.js scripts/install-playwright.ps1
var Scripts embed.FS

// LaunchServerJS returns the launch-server.js content
func LaunchServerJS() ([]byte, error) {
	return Scripts.ReadFile("scripts/launch-server.js")
}

// InstallPlaywrightPS1 returns the install-playwright.ps1 content
func InstallPlaywrightPS1() ([]byte, error) {
	return Scripts.ReadFile("scripts/install-playwright.ps1")
}
