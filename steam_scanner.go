package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// GameScanner handles scanning Steam libraries for games
type GameScanner struct {
	libraries []Library
	parser    *VDFParser
}

// NewGameScanner creates a new game scanner instance
func NewGameScanner(libraries []Library) *GameScanner {
	return &GameScanner{
		libraries: libraries,
		parser:    NewVDFParser(),
	}
}

// ScanAllLibraries scans all Steam libraries for games in parallel
func (gs *GameScanner) ScanAllLibraries() ([]Game, error) {
	var wg sync.WaitGroup
	gamesChan := make(chan []Game, len(gs.libraries))
	errorsChan := make(chan error, len(gs.libraries))

	// Scan each library in parallel
	for _, library := range gs.libraries {
		wg.Add(1)
		go func(lib Library) {
			defer wg.Done()

			games, err := gs.scanLibrary(lib)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️ Error scanning library %s: %v\n", lib.Label, err)
				}
				errorsChan <- fmt.Errorf("library %s: %w", lib.Label, err)
			} else {
				gamesChan <- games
			}
		}(library)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(gamesChan)
		close(errorsChan)
	}()

	// Collect results
	var allGames []Game
	var errors []error

	for games := range gamesChan {
		allGames = append(allGames, games...)
	}

	for err := range errorsChan {
		errors = append(errors, err)
	}

	// Sort games by name for consistent output
	sort.Slice(allGames, func(i, j int) bool {
		return allGames[i].Name < allGames[j].Name
	})

	// Return combined error if any occurred, but still return found games
	if len(errors) > 0 && verbose {
		for _, err := range errors {
			fmt.Printf("⚠️ %v\n", err)
		}
	}

	if verbose {
		fmt.Printf("🎮 Found %d games across all libraries\n", len(allGames))
	}

	return allGames, nil
}

// scanLibrary scans a single Steam library for games
func (gs *GameScanner) scanLibrary(library Library) ([]Game, error) {
	steamAppsPath := filepath.Join(library.Path, "steamapps")

	// Check if steamapps directory exists
	if _, err := os.Stat(steamAppsPath); err != nil {
		return nil, fmt.Errorf("steamapps directory not found: %w", err)
	}

	// Find all appmanifest_*.acf files
	pattern := filepath.Join(steamAppsPath, "appmanifest_*.acf")
	manifests, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding manifest files: %w", err)
	}

	if len(manifests) == 0 {
		if verbose {
			fmt.Printf("📂 No games found in library: %s\n", library.Label)
		}
		return []Game{}, nil
	}

	var games []Game
	commonPath := filepath.Join(steamAppsPath, "common")

	for _, manifestPath := range manifests {
		game, err := gs.parseGameManifest(manifestPath, library, commonPath)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️ Skipping invalid manifest %s: %v\n", filepath.Base(manifestPath), err)
			}
			continue
		}

		// Verify game installation exists
		if !gs.verifyGameInstallation(game.InstallPath) {
			if verbose {
				fmt.Printf("⚠️ Skipping uninstalled game: %s (path: %s)\n", game.Name, game.InstallPath)
			}
			continue
		}

		// Find main executable
		executable, err := gs.FindGameExecutable(game.InstallPath, game.Name)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️ Could not find executable for %s: %v\n", game.Name, err)
			}
			// Still add the game but without executable
			game.Executable = ""
		} else {
			game.Executable = executable
		}

		games = append(games, game)
	}

	if verbose {
		fmt.Printf("📚 Library %s: Found %d games\n", library.Label, len(games))
	}

	return games, nil
}

// parseGameManifest parses a single app manifest file
func (gs *GameScanner) parseGameManifest(manifestPath string, library Library, commonPath string) (Game, error) {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return Game{}, fmt.Errorf("failed to read manifest: %w", err)
	}

	gameInfo, err := gs.parser.ParseAppManifest(content)
	if err != nil {
		return Game{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Convert size on disk to MB
	sizeMB := int64(0)
	if gameInfo.SizeOnDisk != "" {
		if sizeBytes, err := gs.parser.ParseSizeOnDisk(gameInfo.SizeOnDisk); err == nil {
			sizeMB = sizeBytes / (1024 * 1024) // Convert to MB
		}
	}

	// Build install path
	installPath := filepath.Join(commonPath, gameInfo.InstallDir)

	game := Game{
		Name:        gameInfo.Name,
		AppID:       gameInfo.AppID,
		InstallPath: installPath,
		Library:     library.Label,
		SizeMB:      sizeMB,
	}

	return game, nil
}

// verifyGameInstallation checks if the game is actually installed
func (gs *GameScanner) verifyGameInstallation(installPath string) bool {
	if installPath == "" {
		return false
	}

	stat, err := os.Stat(installPath)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

// FindGameExecutable attempts to find the main executable for a game
func (gs *GameScanner) FindGameExecutable(installDir, gameName string) (string, error) {
	if installDir == "" {
		return "", fmt.Errorf("install directory is empty")
	}

	// Common executable patterns to search for
	patterns := gs.generateExecutablePatterns(gameName)

	// Search in the install directory and common subdirectories
	searchPaths := []string{
		installDir,
		filepath.Join(installDir, "Binaries", "Win64"),
		filepath.Join(installDir, "bin"),
		filepath.Join(installDir, "x64"),
		filepath.Join(installDir, "Game", "Binaries", "Win64"),
		filepath.Join(installDir, "Shipping", "Binaries", "Win64"),
	}

	for _, searchPath := range searchPaths {
		if executable := gs.searchExecutableInPath(searchPath, patterns); executable != "" {
			// Return just the filename, not the full path
			return filepath.Base(executable), nil
		}
	}

	// Enhanced fallback: try to find executables that contain parts of the game name
	if executable := gs.findExecutableByGameName(installDir, gameName); executable != "" {
		return filepath.Base(executable), nil
	}

	// Final fallback: find the largest .exe file
	if executable := gs.findLargestExecutable(installDir); executable != "" {
		return filepath.Base(executable), nil
	}

	return "", fmt.Errorf("no executable found")
}

// generateExecutablePatterns creates potential executable names based on game name
func (gs *GameScanner) generateExecutablePatterns(gameName string) []string {
	patterns := []string{}

	// Clean game name for patterns
	cleanName := strings.ToLower(gameName)
	cleanName = strings.ReplaceAll(cleanName, " ", "")
	cleanName = strings.ReplaceAll(cleanName, ":", "")
	cleanName = strings.ReplaceAll(cleanName, "'", "")
	cleanName = strings.ReplaceAll(cleanName, "-", "")
	cleanName = strings.ReplaceAll(cleanName, "™", "")
	cleanName = strings.ReplaceAll(cleanName, "®", "")

	// Generate dynamic patterns based on common naming conventions
	patterns = append(patterns, gs.generateDynamicPatterns(gameName)...)

	// Add common generic patterns
	patterns = append(patterns, "game.exe", "main.exe", "launcher.exe")

	return patterns
}

// searchExecutableInPath searches for executables in a specific path using patterns
func (gs *GameScanner) searchExecutableInPath(searchPath string, patterns []string) string {
	if _, err := os.Stat(searchPath); err != nil {
		return ""
	}

	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return ""
	}

	// First, try exact pattern matches
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := strings.ToLower(entry.Name())
		for _, pattern := range patterns {
			if fileName == strings.ToLower(pattern) {
				return filepath.Join(searchPath, entry.Name())
			}
		}
	}

	return ""
}

// findLargestExecutable finds the largest .exe file as a fallback
func (gs *GameScanner) findLargestExecutable(installDir string) string {
	var largestFile string
	var largestSize int64

	err := filepath.Walk(installDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip if not an executable
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".exe") {
			return nil
		}

		// Skip system and helper executables
		skipPatterns := []string{
			"unins", "setup", "install", "update", "crash", "report",
			"redist", "vcredist", "directx", "dotnet", "unity",
		}

		lowerName := strings.ToLower(info.Name())
		skip := false
		for _, pattern := range skipPatterns {
			if strings.Contains(lowerName, pattern) {
				skip = true
				break
			}
		}

		if skip {
			return nil
		}

		// Check if this is the largest file so far
		if info.Size() > largestSize {
			largestSize = info.Size()
			largestFile = path
		}

		return nil
	})

	if err != nil || largestFile == "" {
		return ""
	}

	// Only return if the file is reasonably large (> 100KB) to avoid tiny helper exes
	if largestSize < 100*1024 {
		return ""
	}

	return largestFile
}

// findExecutableByGameName looks for executables using intelligent matching
func (gs *GameScanner) findExecutableByGameName(installDir, gameName string) string {
	gameWords := strings.Fields(strings.ToLower(gameName))
	if len(gameWords) == 0 {
		return ""
	}

	var bestMatch string
	var bestScore int

	// Generate dynamic patterns to compare against
	dynamicPatterns := gs.generateDynamicPatterns(gameName)
	
	err := filepath.Walk(installDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Only check .exe files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".exe") {
			return nil
		}

		// Skip system and helper executables
		if gs.isSystemExecutable(info.Name()) {
			return nil
		}

		executableName := strings.ToLower(info.Name())
		score := gs.calculateMatchScore(executableName, gameName, gameWords, dynamicPatterns)

		// Update best match if this score is better
		if score > bestScore {
			bestScore = score
			bestMatch = path
		}

		return nil
	})

	if err != nil || bestMatch == "" || bestScore == 0 {
		return ""
	}

	return bestMatch
}

// isSystemExecutable checks if an executable should be skipped
func (gs *GameScanner) isSystemExecutable(filename string) bool {
	lowerName := strings.ToLower(filename)
	
	skipPatterns := []string{
		"unins", "setup", "install", "update", "crash", "report",
		"redist", "vcredist", "directx", "dotnet", "unity", "ue4", "ue5",
		"prerequisites", "support", "helper", "service", "daemon",
		"config", "settings", "options", "benchmark", "test",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}
	
	return false
}

// calculateMatchScore calculates how well an executable name matches the game
func (gs *GameScanner) calculateMatchScore(execName, gameName string, gameWords, patterns []string) int {
	score := 0
	execNameWithoutExt := strings.TrimSuffix(execName, ".exe")
	
	// Score 1: Exact match against any generated pattern (highest priority)
	for _, pattern := range patterns {
		patternWithoutExt := strings.TrimSuffix(pattern, ".exe")
		if execNameWithoutExt == patternWithoutExt {
			return 1000 // Highest score for exact pattern match
		}
	}
	
	// Score 2: Exact match of full game name
	if execNameWithoutExt == strings.ToLower(gameName) {
		return 900
	}
	
	// Score 3: Contains all significant words from game name
	significantWords := gs.getSignificantWords(gameWords)
	containsAllWords := true
	wordScore := 0
	
	for _, word := range significantWords {
		if strings.Contains(execNameWithoutExt, word) {
			wordScore += len(word) * 10 // Higher score for longer words
		} else {
			containsAllWords = false
		}
	}
	
	if containsAllWords && len(significantWords) > 0 {
		score += 800 + wordScore
	} else {
		score += wordScore
	}
	
	// Score 4: Starts with first word of game name
	if len(gameWords) > 0 && len(gameWords[0]) > 2 {
		if strings.HasPrefix(execNameWithoutExt, gameWords[0]) {
			score += 100
		}
	}
	
	// Score 5: Penalty for very short names (likely helpers)
	if len(execNameWithoutExt) < 4 {
		score -= 50
	}
	
	return score
}

// getSignificantWords filters out common words that aren't useful for matching
func (gs *GameScanner) getSignificantWords(words []string) []string {
	var significant []string
	
	for _, word := range words {
		// Skip very short words and common words
		if len(word) > 2 && !gs.isCommonWord(word) {
			significant = append(significant, word)
		}
	}
	
	return significant
}

// generateDynamicPatterns creates executable patterns based on common naming conventions
func (gs *GameScanner) generateDynamicPatterns(gameName string) []string {
	var patterns []string
	
	// Clean the game name
	originalName := strings.ToLower(gameName)
	cleanName := gs.cleanGameName(gameName)
	
	// Pattern 1: Exact game name (handles "Schedule I.exe")
	patterns = append(patterns, originalName+".exe")
	
	// Pattern 2: Underscore version (handles "hollow_knight.exe")
	underscoreName := strings.ReplaceAll(originalName, " ", "_")
	patterns = append(patterns, underscoreName+".exe")
	
	// Pattern 3: No spaces (handles "eldenring.exe")
	noSpaceName := strings.ReplaceAll(originalName, " ", "")
	patterns = append(patterns, noSpaceName+".exe")
	
	// Pattern 4: Clean name without special chars
	patterns = append(patterns, cleanName+".exe")
	
	// Pattern 5: First word only (handles "hunt.exe" for "Hunt: Showdown")
	words := strings.Fields(originalName)
	if len(words) > 0 {
		firstWord := words[0]
		// Remove colons and special chars from first word
		firstWord = strings.Trim(firstWord, ":;-")
		if len(firstWord) > 2 {
			patterns = append(patterns, firstWord+".exe")
		}
	}
	
	// Pattern 6: Acronym from first letters (handles "gtav.exe" for "Grand Theft Auto V")
	acronym := gs.generateAcronym(gameName)
	if len(acronym) > 1 {
		patterns = append(patterns, acronym+".exe")
	}
	
	// Pattern 7: Handle numbered sequels (handles "cs2.exe" for "Counter-Strike 2")
	if numberedPattern := gs.handleNumberedGame(gameName); numberedPattern != "" {
		patterns = append(patterns, numberedPattern+".exe")
	}
	
	// Pattern 8: Handle "Win64" suffix for games like RimWorld
	patterns = append(patterns, cleanName+"win64.exe")
	patterns = append(patterns, noSpaceName+"win64.exe")
	
	// Pattern 9: Handle common suffixes
	suffixes := []string{"game", "client", "main"}
	for _, suffix := range suffixes {
		patterns = append(patterns, cleanName+suffix+".exe")
		patterns = append(patterns, noSpaceName+suffix+".exe")
	}
	
	// Remove duplicates
	return gs.removeDuplicatePatterns(patterns)
}

// cleanGameName removes special characters and normalizes the name
func (gs *GameScanner) cleanGameName(gameName string) string {
	cleanName := strings.ToLower(gameName)
	
	// Remove common special characters
	replacements := map[string]string{
		":":  "",
		"'":  "",
		"-":  "",
		"™":  "",
		"®":  "",
		"©":  "",
		".":  "",
		",":  "",
		"!":  "",
		"?":  "",
		"&":  "and",
		" ":  "",
	}
	
	for old, new := range replacements {
		cleanName = strings.ReplaceAll(cleanName, old, new)
	}
	
	return cleanName
}

// generateAcronym creates an acronym from the first letters of each word
func (gs *GameScanner) generateAcronym(gameName string) string {
	words := strings.Fields(strings.ToLower(gameName))
	var acronym strings.Builder
	
	for _, word := range words {
		// Skip common words that usually don't appear in acronyms
		skipWords := map[string]bool{
			"the": true, "of": true, "and": true, "a": true, "an": true,
			"in": true, "on": true, "at": true, "to": true, "for": true,
			"with": true, "by": true,
		}
		
		if !skipWords[word] && len(word) > 0 {
			acronym.WriteByte(word[0])
		}
	}
	
	return acronym.String()
}

// handleNumberedGame handles games with numbers/sequels dynamically
func (gs *GameScanner) handleNumberedGame(gameName string) string {
	lowerName := strings.ToLower(gameName)
	words := strings.Fields(lowerName)
	
	// Look for numbers in the game name
	for i, word := range words {
		// Check if word is a number (including roman numerals)
		isNumber := gs.isNumber(word)
		
		if isNumber && i > 0 {
			// Create abbreviation from previous words + number
			var abbrev strings.Builder
			for j := 0; j < i; j++ {
				if len(words[j]) > 0 && !gs.isCommonWord(words[j]) {
					abbrev.WriteByte(words[j][0])
				}
			}
			if abbrev.Len() > 1 {
				return abbrev.String() + word
			}
		}
		
		// Special case: if the word contains digits (like "2042")
		if gs.containsDigits(word) && len(word) > 1 {
			// Try to combine with first letters of previous words
			var abbrev strings.Builder
			for j := 0; j < i; j++ {
				if len(words[j]) > 0 && !gs.isCommonWord(words[j]) {
					abbrev.WriteByte(words[j][0])
				}
			}
			if abbrev.Len() > 0 {
				return abbrev.String() + word
			}
		}
	}
	
	return ""
}

// isNumber checks if a string represents a number (including roman numerals)
func (gs *GameScanner) isNumber(word string) bool {
	// Check regular numbers
	if len(word) == 1 && word >= "0" && word <= "9" {
		return true
	}
	
	// Check roman numerals
	romanNumerals := []string{"i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix", "x"}
	for _, roman := range romanNumerals {
		if word == roman {
			return true
		}
	}
	
	return false
}

// containsDigits checks if a string contains any digits
func (gs *GameScanner) containsDigits(word string) bool {
	for _, char := range word {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

// isCommonWord checks if a word is commonly skipped in abbreviations
func (gs *GameScanner) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "of": true, "and": true, "a": true, "an": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"with": true, "by": true, "or": true, "but": true, "is": true,
	}
	return commonWords[word]
}

// removeDuplicatePatterns removes duplicate patterns from the slice
func (gs *GameScanner) removeDuplicatePatterns(patterns []string) []string {
	seen := make(map[string]bool)
	var unique []string
	
	for _, pattern := range patterns {
		if !seen[pattern] {
			seen[pattern] = true
			unique = append(unique, pattern)
		}
	}
	
	return unique
}
