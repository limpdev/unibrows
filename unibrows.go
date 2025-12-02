// Package unibrows provides simple access to browser cookies and bookmarks
// across Chrome, Edge, and other Chromium-based browsers.
//
// Basic usage:
//
//	import "github.com/limpdev/unibrows"
//
//	// Extract all data from Chrome
//	data, err := unibrows.Chrome()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Just get cookies
//	cookies, err := unibrows.ChromeCookies()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Filter cookies by domain
//	githubCookies := cookies.ForDomain("github.com")
package unibrows

import (
	"fmt"
	"runtime"
	"time"
)

// Cookie represents a browser cookie with all relevant metadata
type Cookie struct {
	Host       string    `json:"host"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Value      string    `json:"value"`
	IsSecure   bool      `json:"is_secure"`
	IsHTTPOnly bool      `json:"is_http_only"`
	SameSite   int       `json:"same_site"`
	CreateDate time.Time `json:"create_date"`
	ExpireDate time.Time `json:"expire_date"`
}

// Bookmark represents a browser bookmark
type Bookmark struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Folder    string    `json:"folder"`
	DateAdded time.Time `json:"date_added"`
}

// BrowserData contains all extracted browser data
type BrowserData struct {
	Browser   string
	Profile   string
	Cookies   Cookies
	Bookmarks Bookmarks
}

// Cookies is a slice of Cookie with helper methods
type Cookies []Cookie

// ForDomain returns all cookies matching the given domain
func (c Cookies) ForDomain(domain string) Cookies {
	var result Cookies
	for _, cookie := range c {
		if cookie.Host == domain || cookie.Host == "."+domain {
			result = append(result, cookie)
		}
	}
	return result
}

// ForDomainSuffix returns all cookies for domains ending with the suffix
func (c Cookies) ForDomainSuffix(suffix string) Cookies {
	var result Cookies
	for _, cookie := range c {
		if len(cookie.Host) >= len(suffix) &&
			cookie.Host[len(cookie.Host)-len(suffix):] == suffix {
			result = append(result, cookie)
		}
	}
	return result
}

// AsMap returns cookies as a map[name]value for easy lookup
func (c Cookies) AsMap() map[string]string {
	m := make(map[string]string, len(c))
	for _, cookie := range c {
		m[cookie.Name] = cookie.Value
	}
	return m
}

// Bookmarks is a slice of Bookmark with helper methods
type Bookmarks []Bookmark

// InFolder returns all bookmarks in the specified folder
func (b Bookmarks) InFolder(folder string) Bookmarks {
	var result Bookmarks
	for _, bookmark := range b {
		if bookmark.Folder == folder {
			result = append(result, bookmark)
		}
	}
	return result
}

// Chrome extracts all data from Google Chrome's default profile
func Chrome() (*BrowserData, error) {
	return extract("chrome")
}

// ChromeCookies extracts only cookies from Chrome
func ChromeCookies() (Cookies, error) {
	data, err := Chrome()
	if err != nil {
		return nil, err
	}
	return data.Cookies, nil
}

// ChromeBookmarks extracts only bookmarks from Chrome
func ChromeBookmarks() (Bookmarks, error) {
	data, err := Chrome()
	if err != nil {
		return nil, err
	}
	return data.Bookmarks, nil
}

// Edge extracts all data from Microsoft Edge's default profile
func Edge() (*BrowserData, error) {
	return extract("edge")
}

// EdgeCookies extracts only cookies from Edge
func EdgeCookies() (Cookies, error) {
	data, err := Edge()
	if err != nil {
		return nil, err
	}
	return data.Cookies, nil
}

// EdgeBookmarks extracts only bookmarks from Edge
func EdgeBookmarks() (Bookmarks, error) {
	data, err := Edge()
	if err != nil {
		return nil, err
	}
	return data.Bookmarks, nil
}

// Extract extracts data from a specific browser and optional profile path
// Supported browsers: "chrome", "edge", "brave", "opera"
func Extract(browserName string, profilePath ...string) (*BrowserData, error) {
	if len(profilePath) > 0 {
		return extractCustomProfile(browserName, profilePath[0])
	}
	return extract(browserName)
}

// IsSupported returns true if the browser is supported on this OS
func IsSupported(browserName string) bool {
	_, ok := browserConfigs[runtime.GOOS][browserName]
	return ok
}

// SupportedBrowsers returns a list of browsers supported on this OS
func SupportedBrowsers() []string {
	configs := browserConfigs[runtime.GOOS]
	browsers := make([]string, 0, len(configs))
	for name := range configs {
		browsers = append(browsers, name)
	}
	return browsers
}

// Helper functions (internal)

func extract(browserName string) (*BrowserData, error) {
	browser, err := getBrowser(browserName)
	if err != nil {
		return nil, err
	}
	return browser.extract()
}

func extractCustomProfile(browserName, profilePath string) (*BrowserData, error) {
	browser, err := getBrowserWithProfile(browserName, profilePath)
	if err != nil {
		return nil, err
	}
	return browser.extract()
}

// Error types for better error handling

type ErrUnsupportedOS struct {
	OS string
}

func (e ErrUnsupportedOS) Error() string {
	return fmt.Sprintf("operating system not supported: %s", e.OS)
}

type ErrUnsupportedBrowser struct {
	Browser string
	OS      string
}

func (e ErrUnsupportedBrowser) Error() string {
	return fmt.Sprintf("browser %s not supported on %s", e.Browser, e.OS)
}

type ErrProfileNotFound struct {
	Browser string
	Path    string
}

func (e ErrProfileNotFound) Error() string {
	return fmt.Sprintf("profile for %s not found at %s", e.Browser, e.Path)
}

type ErrDecryption struct {
	Browser string
	Reason  string
}

func (e ErrDecryption) Error() string {
	return fmt.Sprintf("failed to decrypt %s data: %s", e.Browser, e.Reason)
}
