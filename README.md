# Unibrows

A lightweight Go package for extracting cookies and bookmarks from Chromium-based browsers across Windows, macOS, and Linux.

## Features

- **Cross-platform support**: Works on Windows, macOS, and Linux
- **Multiple browsers**: Chrome, Edge, Brave, Opera, Vivaldi, and more
- **Simple API**: Extract all data or just what you need
- **Secure decryption**: Handles browser-specific encryption (DPAPI on Windows, Keychain on macOS)
- **Helper methods**: Filter cookies by domain, organize bookmarks by folder

## Supported Browsers

| Browser  | Windows | macOS | Linux |
| -------- | ------- | ----- | ----- |
| Chrome   | ✓       | ✓     | ✓     |
| Edge     | ✓       | ✓     | ✓     |
| Brave    | ✓       | ✓     | ✓     |
| Opera    | ✓       | ✓     | -     |
| Vivaldi  | ✓       | ✓     | -     |
| Chromium | -       | -     | ✓     |

## Installation

```bash
go get github.com/limpdev/unibrows
```

## Quick Start

### Extract Everything from Chrome

```go
package main

import (
    "fmt"
    "log"

    "github.com/limpdev/unibrows"
)

func main() {
    data, err := unibrows.Chrome()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Browser: %s\n", data.Browser)
    fmt.Printf("Profile: %s\n", data.Profile)
    fmt.Printf("Cookies: %d\n", len(data.Cookies))
    fmt.Printf("Bookmarks: %d\n", len(data.Bookmarks))
}
```

### Extract Only Cookies

```go
cookies, err := unibrows.ChromeCookies()
if err != nil {
    log.Fatal(err)
}

for _, cookie := range cookies {
    fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
}
```

### Extract Only Bookmarks

```go
bookmarks, err := unibrows.ChromeBookmarks()
if err != nil {
    log.Fatal(err)
}

for _, bookmark := range bookmarks {
    fmt.Printf("[%s] %s - %s\n", bookmark.Folder, bookmark.Name, bookmark.URL)
}
```

## Working with Cookies

### Filter by Domain

```go
cookies, _ := unibrows.ChromeCookies()

// Get cookies for a specific domain
githubCookies := cookies.ForDomain("github.com")

// Get cookies for all subdomains
googleCookies := cookies.ForDomainSuffix("google.com")
```

### Convert to Map for Easy Lookup

```go
cookies, _ := unibrows.ChromeCookies()
cookieMap := cookies.AsMap()

// Quick access by name
sessionID := cookieMap["session_id"]
authToken := cookieMap["auth_token"]
```

### Using Cookies in HTTP Requests

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/limpdev/unibrows"
)

func main() {
    // Extract cookies from Chrome
    cookies, err := unibrows.ChromeCookies()
    if err != nil {
        panic(err)
    }

    // Filter for specific domain
    githubCookies := cookies.ForDomain("github.com")

    // Create HTTP client
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "https://github.com/api/user", nil)

    // Add cookies to request
    for _, cookie := range githubCookies {
        req.AddCookie(&http.Cookie{
            Name:  cookie.Name,
            Value: cookie.Value,
        })
    }

    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    fmt.Printf("Response status: %s\n", resp.Status)
}
```

## Passing Cookies to Another Script

### Example 1: Export to JSON

```go
package main

import (
    "encoding/json"
    "os"

    "github.com/limpdev/unibrows"
)

func main() {
    cookies, err := unibrows.ChromeCookies()
    if err != nil {
        panic(err)
    }

    // Filter for specific domain
    targetCookies := cookies.ForDomain("example.com")

    // Export to JSON file
    file, _ := os.Create("cookies.json")
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    encoder.Encode(targetCookies)
}
```

### Example 2: Pass via HTTP API

```go
package main

import (
    "encoding/json"
    "net/http"

    "github.com/limpdev/unibrows"
)

func cookieHandler(w http.ResponseWriter, r *http.Request) {
    domain := r.URL.Query().Get("domain")

    cookies, err := unibrows.ChromeCookies()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Filter by domain if specified
    if domain != "" {
        cookies = cookies.ForDomain(domain)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(cookies)
}

func main() {
    http.HandleFunc("/cookies", cookieHandler)
    http.ListenAndServe(":8080", nil)
}
```

### Example 3: Environment Variable Export

```go
package main

import (
    "fmt"
    "os"

    "github.com/limpdev/unibrows"
)

func main() {
    cookies, _ := unibrows.ChromeCookies()
    cookieMap := cookies.ForDomain("api.example.com").AsMap()

    // Set as environment variables
    if token, ok := cookieMap["auth_token"]; ok {
        os.Setenv("API_AUTH_TOKEN", token)
    }

    if session, ok := cookieMap["session_id"]; ok {
        os.Setenv("API_SESSION_ID", session)
    }

    // Now execute another script that reads these env vars
    // cmd := exec.Command("./your-script")
    // cmd.Env = os.Environ()
    // cmd.Run()
}
```

## Working with Multiple Browsers

```go
package main

import (
    "fmt"

    "github.com/limpdev/unibrows"
)

func main() {
    browsers := []string{"chrome", "edge", "brave"}

    for _, browserName := range browsers {
        if !unibrows.IsSupported(browserName) {
            fmt.Printf("%s not supported on this OS\n", browserName)
            continue
        }

        data, err := unibrows.Extract(browserName)
        if err != nil {
            fmt.Printf("Failed to extract from %s: %v\n", browserName, err)
            continue
        }

        fmt.Printf("%s: %d cookies, %d bookmarks\n",
            data.Browser, len(data.Cookies), len(data.Bookmarks))
    }
}
```

## Custom Profile Paths

```go
// Extract from a specific profile
customPath := "/path/to/browser/profile"
data, err := unibrows.Extract("chrome", customPath)
```

## Data Structures

### Cookie

```go
type Cookie struct {
    Host       string    // Domain (e.g., ".github.com")
    Path       string    // Path (e.g., "/")
    Name       string    // Cookie name
    Value      string    // Decrypted cookie value
    IsSecure   bool      // HTTPS only
    IsHTTPOnly bool      // Not accessible via JavaScript
    SameSite   int       // SameSite attribute (0, 1, 2)
    CreateDate time.Time // When cookie was created
    ExpireDate time.Time // When cookie expires
}
```

### Bookmark

```go
type Bookmark struct {
    ID        string    // Unique bookmark ID
    Name      string    // Bookmark title
    URL       string    // Bookmark URL
    Folder    string    // Folder path (e.g., "bookmark_bar/Work")
    DateAdded time.Time // When bookmark was added
}
```

## Error Handling

```go
data, err := unibrows.Chrome()
if err != nil {
    switch e := err.(type) {
    case unibrows.ErrUnsupportedOS:
        fmt.Printf("OS not supported: %s\n", e.OS)
    case unibrows.ErrProfileNotFound:
        fmt.Printf("Profile not found: %s\n", e.Path)
    case unibrows.ErrDecryption:
        fmt.Printf("Decryption failed: %s\n", e.Reason)
    default:
        fmt.Printf("Unknown error: %v\n", err)
    }
}
```

## Platform-Specific Notes

### Windows

- Uses DPAPI for decryption
- Requires browser to be closed for reliable extraction

### macOS

- Uses Keychain for decryption
- May require user permission to access Keychain

### Linux

- Uses hardcoded encryption key (v10/v11)
- No additional permissions required

## Practical Use Cases

1. **Session Management**: Transfer authenticated sessions between tools
2. **Testing**: Extract production cookies for integration testing
3. **Migration**: Move browser data between systems
4. **Automation**: Authenticate automated scripts using existing browser sessions
5. **Analysis**: Audit cookies for security or privacy analysis

## License

MIT

## Recognition

This repo is just for funsies, and is heavily inspired by [Hack Browser Data](https://github.com/moonD4rk/HackBrowserData), which also declares a license of MIT. If you are a would-be contributor, **go there please**. This is a personal project for my own development/education.
