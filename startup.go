package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const launchAgentLabel = "com.liamg.tyop"

var launchAgentPlist = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.liamg.tyop</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`))

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func isLaunchAtLoginEnabled() bool {
	_, err := os.Stat(launchAgentPath())
	return err == nil
}

func enableLaunchAtLogin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return launchAgentPlist.Execute(f, exe)
}

func disableLaunchAtLogin() error {
	path := launchAgentPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}
