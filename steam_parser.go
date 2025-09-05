package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VDFParser handles parsing of Valve Data Format (VDF) and ACF files
type VDFParser struct{}

// LibraryInfo represents Steam library information from libraryfolders.vdf
type LibraryInfo struct {
	Path         string            `json:"path"`
	Label        string            `json:"label"`
	Mounted      string            `json:"mounted"`
	ContentStats map[string]string `json:"contentStatsId,omitempty"`
}

// GameInfo represents game information from appmanifest ACF files
type GameInfo struct {
	AppID           string `json:"appid"`
	Universe        string `json:"universe"`
	Name            string `json:"name"`
	StateFlags      string `json:"StateFlags"`
	LastUpdated     string `json:"LastUpdated"`
	UpdateResult    string `json:"UpdateResult"`
	SizeOnDisk      string `json:"SizeOnDisk"`
	BuildID         string `json:"buildid"`
	LastOwner       string `json:"LastOwner"`
	BytesToDownload string `json:"BytesToDownload"`
	BytesDownloaded string `json:"BytesDownloaded"`
	InstallDir      string `json:"installdir"`
}

// NewVDFParser creates a new VDF parser instance
func NewVDFParser() *VDFParser {
	return &VDFParser{}
}

// ParseLibraryFolders parses the libraryfolders.vdf file
func (p *VDFParser) ParseLibraryFolders(content []byte) (map[string]LibraryInfo, error) {
	libraries := make(map[string]LibraryInfo)


	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	var currentLibrary *LibraryInfo
	var currentKey string
	var inLibrarySection bool
	var braceLevel int

	// Regex patterns for parsing
	keyValuePattern := regexp.MustCompile(`^\s*"([^"]+)"\s*"([^"]*)"`)
	bracePattern := regexp.MustCompile(`[{}]`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}


		// Count braces to track nesting
		braces := bracePattern.FindAllString(line, -1)
		for _, brace := range braces {
			if brace == "{" {
				braceLevel++
			} else {
				braceLevel--
			}
		}

		// Check for standalone numeric key (like "0" or "1") - library section headers
		keyOnlyPattern := regexp.MustCompile(`^\s*"([^"]+)"\s*$`)
		if braceLevel == 1 && keyOnlyPattern.MatchString(line) {
			matches := keyOnlyPattern.FindStringSubmatch(line)
			key := matches[1]
			
			if _, err := strconv.Atoi(key); err == nil {
				// This is a library section
				currentKey = key
				currentLibrary = &LibraryInfo{}
				inLibrarySection = true
				continue
			}
		}

		// Check if we're parsing key-value pairs within a library section
		if keyValuePattern.MatchString(line) {
			matches := keyValuePattern.FindStringSubmatch(line)
			key, value := matches[1], matches[2]

			// If we're in a library section, parse the key-value pairs
			if inLibrarySection && currentLibrary != nil {
				switch strings.ToLower(key) {
				case "path":
					currentLibrary.Path = value
				case "label":
					currentLibrary.Label = value
				case "mounted":
					currentLibrary.Mounted = value
				case "contentstatsdid":
					if currentLibrary.ContentStats == nil {
						currentLibrary.ContentStats = make(map[string]string)
					}
					currentLibrary.ContentStats["contentStatsId"] = value
				}
			}
		} else if strings.Contains(line, "}") && inLibrarySection && braceLevel == 1 {
			// End of current library section
			if currentLibrary != nil && currentKey != "" {
				libraries[currentKey] = *currentLibrary
			}
			currentLibrary = nil
			currentKey = ""
			inLibrarySection = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning library folders: %w", err)
	}

	return libraries, nil
}

// ParseAppManifest parses an ACF (App Cache File) manifest
func (p *VDFParser) ParseAppManifest(content []byte) (GameInfo, error) {
	var gameInfo GameInfo

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	keyValuePattern := regexp.MustCompile(`^\s*"([^"]+)"\s*"([^"]*)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if keyValuePattern.MatchString(line) {
			matches := keyValuePattern.FindStringSubmatch(line)
			key, value := matches[1], matches[2]

			switch strings.ToLower(key) {
			case "appid":
				gameInfo.AppID = value
			case "universe":
				gameInfo.Universe = value
			case "name":
				gameInfo.Name = value
			case "stateflags":
				gameInfo.StateFlags = value
			case "lastupdated":
				gameInfo.LastUpdated = value
			case "updateresult":
				gameInfo.UpdateResult = value
			case "sizeondisk":
				gameInfo.SizeOnDisk = value
			case "buildid":
				gameInfo.BuildID = value
			case "lastowner":
				gameInfo.LastOwner = value
			case "bytestodownload":
				gameInfo.BytesToDownload = value
			case "bytesdownloaded":
				gameInfo.BytesDownloaded = value
			case "installdir":
				gameInfo.InstallDir = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return gameInfo, fmt.Errorf("error scanning app manifest: %w", err)
	}

	// Validate required fields
	if gameInfo.AppID == "" || gameInfo.Name == "" || gameInfo.InstallDir == "" {
		return gameInfo, fmt.Errorf("missing required fields in app manifest")
	}

	return gameInfo, nil
}

// ParseSizeOnDisk converts the SizeOnDisk string to bytes
func (p *VDFParser) ParseSizeOnDisk(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return size, nil
}
