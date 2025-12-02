package unibrows

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type browserConfig struct {
	name        string
	profilePath string
	storageName string // macOS keychain name
}

type browser interface {
	extract() (*BrowserData, error)
}

var browserConfigs = map[string]map[string]browserConfig{}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	switch runtime.GOOS {
	case "windows":
		browserConfigs["windows"] = map[string]browserConfig{
			"chrome": {
				name:        "Google Chrome",
				profilePath: filepath.Join(homeDir, "AppData", "Local", "Google", "Chrome", "User Data", "Default"),
			},
			"edge": {
				name:        "Microsoft Edge",
				profilePath: filepath.Join(homeDir, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default"),
			},
			"brave": {
				name:        "Brave",
				profilePath: filepath.Join(homeDir, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data", "Default"),
			},
			"opera": {
				name:        "Thorium",
				profilePath: filepath.Join(homeDir, "AppData", "Local", "Thorium", "User Data", "Default"),
			},
			"vivaldi": {
				name:        "Vivaldi",
				profilePath: filepath.Join(homeDir, "AppData", "Local", "Vivaldi", "User Data", "Default"),
			},
		}

	case "darwin":
		browserConfigs["darwin"] = map[string]browserConfig{
			"chrome": {
				name:        "Google Chrome",
				profilePath: filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default"),
				storageName: "Chrome Safe Storage",
			},
			"edge": {
				name:        "Microsoft Edge",
				profilePath: filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge", "Default"),
				storageName: "Microsoft Edge Safe Storage",
			},
			"brave": {
				name:        "Brave",
				profilePath: filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser", "Default"),
				storageName: "Brave Safe Storage",
			},
			"opera": {
				name:        "Opera",
				profilePath: filepath.Join(homeDir, "Library", "Application Support", "com.operasoftware.Opera"),
				storageName: "Opera Safe Storage",
			},
			"vivaldi": {
				name:        "Vivaldi",
				profilePath: filepath.Join(homeDir, "Library", "Application Support", "Vivaldi", "Default"),
				storageName: "Vivaldi Safe Storage",
			},
		}

	case "linux":
		browserConfigs["linux"] = map[string]browserConfig{
			"chrome": {
				name:        "Google Chrome",
				profilePath: filepath.Join(homeDir, ".config", "google-chrome", "Default"),
			},
			"chromium": {
				name:        "Chromium",
				profilePath: filepath.Join(homeDir, ".config", "chromium", "Default"),
			},
			"brave": {
				name:        "Brave",
				profilePath: filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser", "Default"),
			},
			"edge": {
				name:        "Microsoft Edge",
				profilePath: filepath.Join(homeDir, ".config", "microsoft-edge", "Default"),
			},
		}
	}
}

func getBrowser(browserName string) (browser, error) {
	configs, ok := browserConfigs[runtime.GOOS]
	if !ok {
		return nil, ErrUnsupportedOS{OS: runtime.GOOS}
	}

	config, ok := configs[browserName]
	if !ok {
		return nil, ErrUnsupportedBrowser{Browser: browserName, OS: runtime.GOOS}
	}

	if !isDirExists(config.profilePath) {
		return nil, ErrProfileNotFound{Browser: config.name, Path: config.profilePath}
	}

	// Currently only support Chromium-based browsers
	return newChromium(config.name, config.profilePath, config.storageName), nil
}

func getBrowserWithProfile(browserName, profilePath string) (browser, error) {
	configs, ok := browserConfigs[runtime.GOOS]
	if !ok {
		return nil, ErrUnsupportedOS{OS: runtime.GOOS}
	}

	config, ok := configs[browserName]
	if !ok {
		return nil, ErrUnsupportedBrowser{Browser: browserName, OS: runtime.GOOS}
	}

	if !isDirExists(profilePath) {
		return nil, ErrProfileNotFound{Browser: config.name, Path: profilePath}
	}

	return newChromium(config.name, profilePath, config.storageName), nil
}

// Utility functions

func isDirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}
	return nil
}
