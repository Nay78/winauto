package win

import (
	"fmt"
	"strconv"
	"strings"
)

// PowerShellCommand wraps an inline script in the standard powershell invocation.
func PowerShellCommand(script string) string {
	return fmt.Sprintf("powershell -NoProfile -Command %s", strconv.Quote(script))
}

// FirewallRuleCheck returns a PowerShell command that evaluates whether a firewall rule is enabled.
func FirewallRuleCheck(displayName string) string {
	escaped := strings.ReplaceAll(displayName, "'", "''")
	script := fmt.Sprintf("$rule = Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue; if ($null -eq $rule) { $false } else { $rule.Enabled -eq 'True' }", escaped)
	return PowerShellCommand(script)
}

// PortListeningCheck returns a PowerShell command that checks whether a TCP listener exists on the given port.
func PortListeningCheck(port int) string {
	script := fmt.Sprintf("if (Get-NetTCPConnection -LocalPort %d -State Listen -ErrorAction SilentlyContinue) { $true } else { $false }", port)
	return PowerShellCommand(script)
}

// DesktopUnlockedCheck returns a PowerShell command that treats the absence of LogonUI as an unlocked desktop.
func DesktopUnlockedCheck() string {
	script := "if (Get-Process -Name LogonUI -ErrorAction SilentlyContinue) { $false } else { $true }"
	return PowerShellCommand(script)
}
