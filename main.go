package main

import (
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/getlantern/systray"
)

//go:embed icon.png
var iconData []byte

func main() {
	if !acquireLock() {
		os.Exit(0)
	}
	cfg := loadConfig()
	systray.Run(func() { onReady(cfg) }, onExit)
}

func onReady(cfg *Config) {
	systray.SetIcon(iconData)
	systray.SetTooltip("tyop — fix typos with " + cfg.Hotkey)

	// Accessibility check — show warning item if not granted, re-check on click.
	if !accessibilityGranted() {
		mAX := systray.AddMenuItem("⚠️ Grant Accessibility Access and Restart", "Open System Settings to grant access, then restart tyop")
		systray.AddSeparator()
		go func() {
			for range mAX.ClickedCh {
				openAccessibilityPrefs()
				// Re-check after a short delay — user may have just granted it
				time.Sleep(2 * time.Second)
				if accessibilityGranted() {
					mAX.Hide()
				}
			}
		}()
	}

	// Enabled toggle
	mEnabled := systray.AddMenuItemCheckbox("Enabled", "Enable/disable tyop", cfg.Enabled)

	systray.AddSeparator()

	// Locale submenu
	mLocaleGB := systray.AddMenuItemCheckbox("English (UK)", "en-gb", cfg.Locale == EnGB)
	mLocaleUS := systray.AddMenuItemCheckbox("English (US)", "en-us", cfg.Locale == EnUS)

	systray.AddSeparator()

	// Hotkey submenu
	mHotkeyLabel := systray.AddMenuItem("Hotkey: "+cfg.Hotkey, "Current hotkey")
	mHotkeyLabel.Disable()
	hotkeys := []string{"Ctrl+.", "Ctrl+,", "Ctrl+;"}
	mHotkeys := make([]*systray.MenuItem, len(hotkeys))
	for i, hk := range hotkeys {
		checked := hk == cfg.Hotkey
		mHotkeys[i] = systray.AddMenuItemCheckbox("  "+hk, hk, checked)
	}

	// Launch at login (enable by default on first run)
	if cfg.LaunchAtLogin == nil {
		enabled := true
		cfg.LaunchAtLogin = &enabled
		_ = enableLaunchAtLogin()
		saveConfig(cfg)
	}
	// Clipboard fallback (enable by default on first run)
	if cfg.ClipboardFallback == nil {
		enabled := true
		cfg.ClipboardFallback = &enabled
		saveConfig(cfg)
	}
	systray.AddSeparator()
	mStartup := systray.AddMenuItemCheckbox("Launch at Login", "Start tyop automatically at login", *cfg.LaunchAtLogin)
	mClipboard := systray.AddMenuItemCheckbox("Clipboard Fallback", "Use clipboard+Cmd+A/V for apps that don't support direct AX writes (e.g. Zed)", *cfg.ClipboardFallback)

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit tyop", "Quit")

	corrector := newCorrector(cfg.Locale)
	if cfg.Enabled {
		startHotkeyListener(cfg.Hotkey)
	}

	go func() {
		cases := make([]reflect.SelectCase, 0, 10+len(mHotkeys))
		addCase := func(ch <-chan struct{}) int {
			idx := len(cases)
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)})
			return idx
		}
		iEnabled := addCase(mEnabled.ClickedCh)
		iLocaleGB := addCase(mLocaleGB.ClickedCh)
		iLocaleUS := addCase(mLocaleUS.ClickedCh)
		iStartup := addCase(mStartup.ClickedCh)
		iClipboard := addCase(mClipboard.ClickedCh)
		iQuit := addCase(mQuit.ClickedCh)
		iHotkeys := make([]int, len(mHotkeys))
		for i, m := range mHotkeys {
			iHotkeys[i] = addCase(m.ClickedCh)
		}

		for {
			chosen, _, _ := reflect.Select(cases)
			switch chosen {
			case iEnabled:
				cfg.Enabled = !cfg.Enabled
				if cfg.Enabled {
					mEnabled.Check()
					startHotkeyListener(cfg.Hotkey)
				} else {
					mEnabled.Uncheck()
					stopHotkeyListener()
				}
				saveConfig(cfg)

			case iLocaleGB:
				cfg.Locale = EnGB
				mLocaleGB.Check()
				mLocaleUS.Uncheck()
				corrector = newCorrector(cfg.Locale)
				saveConfig(cfg)

			case iLocaleUS:
				cfg.Locale = EnUS
				mLocaleUS.Check()
				mLocaleGB.Uncheck()
				corrector = newCorrector(cfg.Locale)
				saveConfig(cfg)

			case iStartup:
				*cfg.LaunchAtLogin = !*cfg.LaunchAtLogin
				if *cfg.LaunchAtLogin {
					mStartup.Check()
					_ = enableLaunchAtLogin()
				} else {
					mStartup.Uncheck()
					_ = disableLaunchAtLogin()
				}
				saveConfig(cfg)

			case iClipboard:
				*cfg.ClipboardFallback = !*cfg.ClipboardFallback
				if *cfg.ClipboardFallback {
					mClipboard.Check()
				} else {
					mClipboard.Uncheck()
				}
				saveConfig(cfg)

			case iQuit:
				systray.Quit()
				return

			default:
				for i, hi := range iHotkeys {
					if chosen == hi {
						cfg.Hotkey = hotkeys[i]
						mHotkeyLabel.SetTitle("Hotkey: " + cfg.Hotkey)
						for j, hm := range mHotkeys {
							if j == i {
								hm.Check()
							} else {
								hm.Uncheck()
							}
						}
						stopHotkeyListener()
						if cfg.Enabled {
							startHotkeyListener(cfg.Hotkey)
						}
						saveConfig(cfg)
						break
					}
				}
			}
		}
	}()

	// Handle corrections on hotkey presses.
	go func() {
		for range hotkeyChannel {
			if !cfg.Enabled {
				continue
			}
			if err := handleCorrection(corrector, *cfg.ClipboardFallback); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	}()
}

func onExit() {}

func handleCorrection(c *Corrector, clipboardFallback bool) error {
	text, usedClipboard, err := readFocusedText(clipboardFallback)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if text == "" {
		return nil
	}
	if len(text) > 1024 {
		return nil
	}

	corrected := c.Correct(text)
	if corrected == text {
		return nil
	}

	return writeFocusedText(corrected, usedClipboard)
}
