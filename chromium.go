package unibrows

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"unibrows/crypto"

	_ "modernc.org/sqlite"
)

type chromium struct {
	name        string
	profilePath string
	storageName string
	masterKey   []byte
}

func newChromium(name, profilePath, storageName string) *chromium {
	return &chromium{
		name:        name,
		profilePath: profilePath,
		storageName: storageName,
	}
}

func (c *chromium) extract() (*BrowserData, error) {
	data := &BrowserData{
		Browser: c.name,
		Profile: c.profilePath,
	}

	// Get master key for decryption
	var err error
	c.masterKey, err = c.getMasterKey()
	if err != nil {
		return nil, ErrDecryption{Browser: c.name, Reason: err.Error()}
	}

	// Extract cookies (continue on error)
	cookies, err := c.extractCookies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not extract cookies for %s: %v\n", c.name, err)
	}
	data.Cookies = cookies

	// Extract bookmarks (continue on error)
	bookmarks, err := c.extractBookmarks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not extract bookmarks for %s: %v\n", c.name, err)
	}
	data.Bookmarks = bookmarks

	return data, nil
}

func (c *chromium) extractCookies() (Cookies, error) {
	cookieDBPath := filepath.Join(c.profilePath, "Network", "Cookies")

	// Check if Cookies file exists (some browsers use different paths)
	if _, err := os.Stat(cookieDBPath); os.IsNotExist(err) {
		// Try alternate path (older Chrome versions)
		cookieDBPath = filepath.Join(c.profilePath, "Cookies")
		if _, err := os.Stat(cookieDBPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("cookies database not found")
		}
	}

	// Copy to temp file to avoid lock issues
	tmpDB := filepath.Join(os.TempDir(), fmt.Sprintf("unibrows_cookies_%d.db", time.Now().UnixNano()))
	defer os.Remove(tmpDB)

	if err := copyFile(cookieDBPath, tmpDB); err != nil {
		return nil, fmt.Errorf("failed to copy cookie database: %w", err)
	}

	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie database: %w", err)
	}
	defer db.Close()

	// Query cookies
	rows, err := db.Query(`
		SELECT 
			host_key, 
			path, 
			name, 
			encrypted_value, 
			is_secure, 
			is_httponly,
			samesite,
			creation_utc, 
			expires_utc
		FROM cookies
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies Cookies
	for rows.Next() {
		var (
			host, path, name     string
			encryptedValue       []byte
			isSecure, isHTTPOnly bool
			sameSite             int
			createUTC, expireUTC int64
		)

		if err := rows.Scan(
			&host, &path, &name, &encryptedValue,
			&isSecure, &isHTTPOnly, &sameSite,
			&createUTC, &expireUTC,
		); err != nil {
			continue // Skip malformed cookies
		}

		// Decrypt the cookie value
		decryptedValue, err := c.decryptValue(encryptedValue)
		if err != nil {
			// Try to use unencrypted value if decryption fails
			decryptedValue = string(encryptedValue)
		}

		cookies = append(cookies, Cookie{
			Host:       host,
			Path:       path,
			Name:       name,
			Value:      decryptedValue,
			IsSecure:   isSecure,
			IsHTTPOnly: isHTTPOnly,
			SameSite:   sameSite,
			CreateDate: chromeTime(createUTC),
			ExpireDate: chromeTime(expireUTC),
		})
	}

	return cookies, nil
}

func (c *chromium) extractBookmarks() (Bookmarks, error) {
	bookmarkPath := filepath.Join(c.profilePath, "Bookmarks")

	data, err := os.ReadFile(bookmarkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bookmarks file: %w", err)
	}

	var bookmarkData struct {
		Roots map[string]json.RawMessage `json:"roots"`
	}

	if err := json.Unmarshal(data, &bookmarkData); err != nil {
		return nil, fmt.Errorf("failed to parse bookmarks JSON: %w", err)
	}

	var bookmarks Bookmarks

	// Parse each root folder (bookmark_bar, other, synced)
	for folderName, folderData := range bookmarkData.Roots {
		if folderName == "sync_transaction_version" || folderName == "meta_info" {
			continue
		}

		var folder bookmarkFolder
		if err := json.Unmarshal(folderData, &folder); err != nil {
			continue
		}

		bookmarks = append(bookmarks, c.parseBookmarkFolder(&folder, folderName)...)
	}

	return bookmarks, nil
}

type bookmarkFolder struct {
	Children []bookmarkNode `json:"children"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
}

type bookmarkNode struct {
	DateAdded string         `json:"date_added"`
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	URL       string         `json:"url"`
	Children  []bookmarkNode `json:"children"`
}

func (c *chromium) parseBookmarkFolder(folder *bookmarkFolder, folderPath string) Bookmarks {
	var bookmarks Bookmarks

	for _, child := range folder.Children {
		bookmarks = append(bookmarks, c.parseBookmarkNode(&child, folderPath)...)
	}

	return bookmarks
}

func (c *chromium) parseBookmarkNode(node *bookmarkNode, folderPath string) Bookmarks {
	var bookmarks Bookmarks

	if node.Type == "url" {
		dateAdded, _ := time.Parse(time.RFC3339, node.DateAdded)
		bookmarks = append(bookmarks, Bookmark{
			ID:        node.ID,
			Name:      node.Name,
			URL:       node.URL,
			Folder:    folderPath,
			DateAdded: dateAdded,
		})
	} else if node.Type == "folder" {
		newPath := folderPath + "/" + node.Name
		for _, child := range node.Children {
			bookmarks = append(bookmarks, c.parseBookmarkNode(&child, newPath)...)
		}
	}

	return bookmarks
}

func (c *chromium) getMasterKey() ([]byte, error) {
	return c.getMasterKeyOS()
}

func (c *chromium) decryptValue(encryptedValue []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	// Try to decrypt with the master key
	decrypted, err := crypto.DecryptWithChromium(c.masterKey, encryptedValue)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// chromeTime converts Chrome's timestamp format to time.Time
// Chrome uses microseconds since Windows epoch (Jan 1, 1601)
func chromeTime(timestamp int64) time.Time {
	if timestamp == 0 {
		return time.Time{}
	}
	// Convert Chrome timestamp (microseconds since 1601) to Unix timestamp
	// 11644473600 seconds between 1601 and 1970
	unixSeconds := (timestamp / 1000000) - 11644473600
	return time.Unix(unixSeconds, 0)
}
