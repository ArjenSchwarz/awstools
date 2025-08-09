# Technical Efficiency Improvements

This document tracks identified efficiency improvements in the awstools codebase.

## [2025-01-27] - Profile Generator Enhancement Efficiency Review

### Issue: Redundant Action Type Counting in Multiple Functions
**Location**: `cmd/sso.go` (lines 270-280, 350-360, 400-410)
**Description**: The `countActionsByType` function is called multiple times with the same data in different display functions, causing redundant O(n) iterations over the same slice.
**Impact**: Minor performance impact - O(n) redundant iterations for each display function call
**Solution**:
```go
// Pre-calculate action counts once and pass to display functions
type ActionCounts struct {
    Replace int
    Skip    int
    Create  int
}

func calculateActionCounts(actions []helpers.ConflictAction) ActionCounts {
    counts := ActionCounts{}
    for _, action := range actions {
        switch action.Action {
        case helpers.ActionReplace:
            counts.Replace++
        case helpers.ActionSkip:
            counts.Skip++
        case helpers.ActionCreate:
            counts.Create++
        }
    }
    return counts
}

// Update display functions to accept pre-calculated counts
func displayConflictResolutionSummary(result *helpers.ProfileGenerationResult, counts ActionCounts) {
    // Use counts.Replace, counts.Skip, etc. instead of calling countActionsByType
}
```
**Trade-offs**: Slightly more complex function signatures but eliminates redundant computation

---

### Issue: Inefficient String Building in FormatProgressMessage
**Location**: `helpers/profile_generator.go` (lines 880-900)
**Description**: The function doesn't pre-allocate string builder capacity and uses inefficient map iteration for details formatting.
**Impact**: Minor performance impact - unnecessary memory reallocations during string building
**Solution**:
```go
func (pg *ProfileGenerator) FormatProgressMessage(phase string, message string, details map[string]any) string {
    // Estimate capacity: phase indicator (4) + message + details
    estimatedSize := 4 + len(message) + len(details)*20
    var msg strings.Builder
    msg.Grow(estimatedSize)

    // Add phase indicator (use map for O(1) lookup instead of switch)
    phaseIcons := map[string]string{
        "validation":         "🔍 ",
        "discovery":          "🔍 ",
        "conflict_detection": "⚠️  ",
        "conflict_resolution": "🔧 ",
        "generation":         "📝 ",
        "success":            "✅ ",
        "error":              "❌ ",
    }
    
    if icon, exists := phaseIcons[phase]; exists {
        msg.WriteString(icon)
    } else {
        msg.WriteString("ℹ️  ")
    }

    msg.WriteString(message)

    // More efficient details formatting
    if len(details) > 0 {
        for key, value := range details {
            msg.WriteString(fmt.Sprintf(" [%s: %v]", key, value))
        }
    }

    return msg.String()
}
```
**Trade-offs**: Slightly more memory usage for the phase icons map, but better performance for repeated calls

---

### Issue: Redundant Profile Name Extraction in Display Functions
**Location**: `cmd/sso.go` (lines 250-270, 320-340)
**Description**: Multiple display functions extract profile names from the same result data structures repeatedly.
**Impact**: Minor performance impact - redundant slice iterations and string operations
**Solution**:
```go
// Pre-extract commonly used data once
type DisplayData struct {
    ProfileNames     []string
    SkippedRoleNames []string
    ActionCounts     ActionCounts
}

func prepareDisplayData(result *helpers.ProfileGenerationResult) DisplayData {
    data := DisplayData{
        ProfileNames:     make([]string, len(result.GeneratedProfiles)),
        SkippedRoleNames: make([]string, len(result.SkippedRoles)),
        ActionCounts:     calculateActionCounts(result.ResolutionActions),
    }
    
    for i, profile := range result.GeneratedProfiles {
        data.ProfileNames[i] = profile.Name
    }
    
    for i, role := range result.SkippedRoles {
        data.SkippedRoleNames[i] = fmt.Sprintf("%s in %s", role.PermissionSetName, role.AccountName)
    }
    
    return data
}
```
**Trade-offs**: Additional memory usage for pre-computed data, but eliminates redundant computations across multiple display functions

---

### Issue: Inefficient Nil Check in FormatProgressMessage
**Location**: `helpers/profile_generator.go` (line 895)
**Description**: The code checks `details != nil && len(details) > 0` but Go's len() function returns 0 for nil maps, making the nil check redundant.
**Impact**: Minimal performance impact - unnecessary nil check
**Solution**:
```go
// Remove redundant nil check
if len(details) > 0 {
    for key, value := range details {
        msg.WriteString(fmt.Sprintf(" [%s: %v]", key, value))
    }
}
```
**Trade-offs**: None - this is a pure improvement

---

### Issue: Potential Memory Allocation Optimization in Profile Generation
**Location**: `helpers/profile_generator.go` (lines 700-750)
**Description**: The `GenerateProfilesForNonConflictedRoles` function doesn't pre-allocate the slice capacity for generated profiles.
**Impact**: Minor performance impact - potential slice reallocations during append operations
**Solution**:
```go
func (pg *ProfileGenerator) GenerateProfilesForNonConflictedRoles(nonConflictedRoles []DiscoveredRole) ([]GeneratedProfile, error) {
    if len(nonConflictedRoles) == 0 {
        return []GeneratedProfile{}, nil
    }

    // Pre-allocate with exact capacity
    generatedProfiles := make([]GeneratedProfile, 0, len(nonConflictedRoles))

    // ... rest of the function remains the same
}
```
**Trade-offs**: None - this is a pure improvement that prevents slice reallocations

---

### Issue: Inefficient Regex Compilation in Config File Parsing
**Location**: `helpers/aws_config_file.go` (lines 150-160, 200-210)
**Description**: Regular expressions for parsing profile sections are compiled on every call to `parseConfigFile` and `parseConfigFileWithRecovery`, causing unnecessary CPU overhead during config file parsing.
**Impact**: Minor to moderate performance impact - regex compilation overhead on every config file parse operation
**Solution**:
```go
// Compile regex patterns once at package level
var (
    profileNameRegex   = regexp.MustCompile(`^\[profile\s+(.+)\]`)
    defaultProfileRegex = regexp.MustCompile(`^\[default\]`)
    ssoSessionRegex    = regexp.MustCompile(`^\[sso-session\s+(.+)\]`)
)

// Use pre-compiled patterns in parsing functions
func (cf *AWSConfigFile) parseConfigFile(file *os.File) error {
    scanner := bufio.NewScanner(file)
    // ... existing code ...
    
    // Use pre-compiled regex patterns
    if matches := profileNameRegex.FindStringSubmatch(line); matches != nil {
        // ... existing logic ...
    }
}
```
**Trade-offs**: Slightly more memory usage for compiled patterns, but significant performance improvement for repeated parsing operations

---

### Issue: Redundant SSO Configuration Resolution in Conflict Detection
**Location**: `helpers/profile_conflict_detector.go` (lines 60-80)
**Description**: The `preResolveSSO()` method iterates through all profiles and resolves SSO configurations, but doesn't handle errors efficiently and may resolve configurations that are never used.
**Impact**: Minor performance impact - unnecessary SSO resolution for profiles that won't be involved in conflicts
**Solution**:
```go
// Lazy resolution with error handling and caching
func (pcd *ProfileConflictDetector) getResolvedSSOConfig(profileName string, profile Profile) (*ResolvedSSOConfig, error) {
    // Check cache first
    if config, exists := pcd.resolvedSSOConfigs[profileName]; exists {
        return config, nil
    }
    
    // Only resolve if profile is SSO
    if !profile.IsSSO() {
        return nil, fmt.Errorf("profile %s is not an SSO profile", profileName)
    }
    
    // Resolve and cache
    resolvedConfig, err := pcd.configFile.ResolveProfileSSOConfig(profile)
    if err != nil {
        return nil, err
    }
    
    pcd.resolvedSSOConfigs[profileName] = resolvedConfig
    return resolvedConfig, nil
}
```
**Trade-offs**: More complex caching logic but avoids unnecessary resolution and provides better error handling

---

### Issue: Inefficient String Concatenation in Profile Generation
**Location**: `helpers/profile_generator_types.go` (lines 100-120)
**Description**: The `ToConfigString()` method uses string concatenation with `fmt.Sprintf` for each line, which creates multiple temporary strings.
**Impact**: Minor performance impact - unnecessary string allocations during profile serialization
**Solution**:
```go
func (gp *GeneratedProfile) ToConfigString() string {
    var config strings.Builder
    // More accurate size estimation: profile name + region + SSO config
    estimatedSize := len(gp.Name) + len(gp.Region) + len(gp.SSOStartURL) + len(gp.SSORegion) + 100
    config.Grow(estimatedSize)
    
    config.WriteString("[profile ")
    config.WriteString(gp.Name)
    config.WriteString("]\nregion = ")
    config.WriteString(gp.Region)
    config.WriteString("\nsso_start_url = ")
    config.WriteString(gp.SSOStartURL)
    config.WriteString("\nsso_region = ")
    config.WriteString(gp.SSORegion)
    config.WriteString("\n")

    if gp.IsLegacy {
        config.WriteString("sso_account_id = ")
        config.WriteString(gp.SSOAccountID)
        config.WriteString("\nsso_role_name = ")
        config.WriteString(gp.SSORoleName)
        config.WriteString("\n")
    } else {
        config.WriteString("sso_session = ")
        config.WriteString(gp.SSOSession)
        config.WriteString("\n")
    }

    return config.String()
}
```
**Trade-offs**: More verbose code but eliminates temporary string allocations from fmt.Sprintf calls

---

### Issue: Inefficient Map Key Generation in Role Filtering
**Location**: `helpers/profile_generator.go` (lines 580-590, 600-610)
**Description**: The `FilterRolesByConflicts` function generates map keys using `fmt.Sprintf` for each role, creating unnecessary string allocations.
**Impact**: Minor performance impact - string allocation overhead during role filtering
**Solution**:
```go
func (pg *ProfileGenerator) FilterRolesByConflicts(discoveredRoles []DiscoveredRole, conflicts []ProfileConflict) (conflictedRoles []DiscoveredRole, nonConflictedRoles []DiscoveredRole) {
    // Create a map of conflicted roles for efficient lookup
    conflictedRoleMap := make(map[string]bool, len(conflicts))
    
    // Pre-allocate string builder for key generation
    var keyBuilder strings.Builder
    keyBuilder.Grow(32) // Typical account ID (12) + separator (1) + role name (~15-20)
    
    for _, conflict := range conflicts {
        keyBuilder.Reset()
        keyBuilder.WriteString(conflict.DiscoveredRole.AccountID)
        keyBuilder.WriteString(":")
        keyBuilder.WriteString(conflict.DiscoveredRole.PermissionSetName)
        conflictedRoleMap[keyBuilder.String()] = true
    }

    // Separate roles based on conflict status
    for _, role := range discoveredRoles {
        keyBuilder.Reset()
        keyBuilder.WriteString(role.AccountID)
        keyBuilder.WriteString(":")
        keyBuilder.WriteString(role.PermissionSetName)
        
        if conflictedRoleMap[keyBuilder.String()] {
            conflictedRoles = append(conflictedRoles, role)
        } else {
            nonConflictedRoles = append(nonConflictedRoles, role)
        }
    }

    return conflictedRoles, nonConflictedRoles
}
```
**Trade-offs**: More complex key generation logic but eliminates fmt.Sprintf allocations

---

---

## [2025-01-27] - Profile Generator Enhancement Efficiency Review (Updated)

### Issue: Redundant Action Type Counting in Display Functions
**Location**: `cmd/sso.go` (lines 368-369, 482-483)
**Description**: The `countActionsByType` function is called multiple times with the same data in different display functions, causing redundant O(n) iterations over the same slice.
**Impact**: Minor performance impact - O(n) redundant iterations for each display function call
**Solution**:
```go
// Pre-calculate action counts once and pass to display functions
type ActionCounts struct {
    Replace int
    Skip    int
    Create  int
}

func calculateActionCounts(actions []helpers.ConflictAction) ActionCounts {
    counts := ActionCounts{}
    for _, action := range actions {
        switch action.Action {
        case helpers.ActionReplace:
            counts.Replace++
        case helpers.ActionSkip:
            counts.Skip++
        case helpers.ActionCreate:
            counts.Create++
        }
    }
    return counts
}

// Update display functions to accept pre-calculated counts
func displayConflictResolutionSummary(result *helpers.ProfileGenerationResult, counts ActionCounts) {
    // Use counts.Replace, counts.Skip, etc. instead of calling countActionsByType
}
```
**Trade-offs**: Slightly more complex function signatures but eliminates redundant computation

---

### Issue: Inefficient String Building in FormatProgressMessage
**Location**: `helpers/profile_generator.go` (lines 1042-1076)
**Description**: The function doesn't pre-allocate string builder capacity and uses inefficient switch statement for phase indicators.
**Impact**: Minor performance impact - unnecessary memory reallocations during string building
**Solution**:
```go
// Pre-compiled phase icons map for O(1) lookup
var phaseIcons = map[string]string{
    "validation":         "🔍 ",
    "discovery":          "🔍 ",
    "conflict_detection": "⚠️  ",
    "conflict_resolution": "🔧 ",
    "generation":         "📝 ",
    "success":            "✅ ",
    "error":              "❌ ",
}

func (pg *ProfileGenerator) FormatProgressMessage(phase string, message string, details map[string]any) string {
    // Estimate capacity: phase indicator (4) + message + details
    estimatedSize := 4 + len(message) + len(details)*20
    var msg strings.Builder
    msg.Grow(estimatedSize)

    // Use map for O(1) lookup instead of switch
    if icon, exists := phaseIcons[phase]; exists {
        msg.WriteString(icon)
    } else {
        msg.WriteString("ℹ️  ")
    }

    msg.WriteString(message)

    // More efficient details formatting
    if len(details) > 0 {
        for key, value := range details {
            msg.WriteString(fmt.Sprintf(" [%s: %v]", key, value))
        }
    }

    return msg.String()
}
```
**Trade-offs**: Slightly more memory usage for the phase icons map, but better performance for repeated calls

---

### Issue: Inefficient String Concatenation in Role Filtering
**Location**: `helpers/profile_generator.go` (lines 845-846, 851-852)
**Description**: The `FilterRolesByConflicts` function generates map keys using `fmt.Sprintf` for each role, creating unnecessary string allocations.
**Impact**: Minor performance impact - string allocation overhead during role filtering
**Solution**:
```go
func (pg *ProfileGenerator) FilterRolesByConflicts(discoveredRoles []DiscoveredRole, conflicts []ProfileConflict) (conflictedRoles []DiscoveredRole, nonConflictedRoles []DiscoveredRole) {
    // Create a map of conflicted roles for efficient lookup
    conflictedRoleMap := make(map[string]bool, len(conflicts))
    
    // Pre-allocate string builder for key generation
    var keyBuilder strings.Builder
    keyBuilder.Grow(32) // Typical account ID (12) + separator (1) + role name (~15-20)
    
    for _, conflict := range conflicts {
        keyBuilder.Reset()
        keyBuilder.WriteString(conflict.DiscoveredRole.AccountID)
        keyBuilder.WriteString(":")
        keyBuilder.WriteString(conflict.DiscoveredRole.PermissionSetName)
        conflictedRoleMap[keyBuilder.String()] = true
    }

    // Separate roles based on conflict status
    for _, role := range discoveredRoles {
        keyBuilder.Reset()
        keyBuilder.WriteString(role.AccountID)
        keyBuilder.WriteString(":")
        keyBuilder.WriteString(role.PermissionSetName)
        
        if conflictedRoleMap[keyBuilder.String()] {
            conflictedRoles = append(conflictedRoles, role)
        } else {
            nonConflictedRoles = append(nonConflictedRoles, role)
        }
    }

    return conflictedRoles, nonConflictedRoles
}
```
**Trade-offs**: More complex key generation logic but eliminates fmt.Sprintf allocations

---

### Issue: Inefficient Regex Compilation in Config File Parsing
**Location**: `helpers/aws_config_file.go` (lines 207-209, 342-344)
**Description**: Regular expressions for parsing profile sections are compiled on every call to `parseConfigFile` and `parseConfigFileWithRecovery`, causing unnecessary CPU overhead during config file parsing.
**Impact**: Minor to moderate performance impact - regex compilation overhead on every config file parse operation
**Solution**:
```go
// Compile regex patterns once at package level
var (
    profileNameRegex   = regexp.MustCompile(`^\[profile\s+(.+)\]$`)
    defaultProfileRegex = regexp.MustCompile(`^\[default\]$`)
    ssoSessionRegex    = regexp.MustCompile(`^\[sso-session\s+(.+)\]$`)
)

// Use pre-compiled patterns in parsing functions
func (cf *AWSConfigFile) parseConfigFile(file *os.File) error {
    scanner := bufio.NewScanner(file)
    // ... existing code ...
    
    // Use pre-compiled regex patterns
    if matches := profileNameRegex.FindStringSubmatch(line); matches != nil {
        // ... existing logic ...
    }
}
```
**Trade-offs**: Slightly more memory usage for compiled patterns, but significant performance improvement for repeated parsing operations

---

## Summary

The profile generator enhancement code is generally well-optimized with good use of:
- Pre-allocated slice capacities in conflict detection
- Cached SSO configurations to avoid repeated resolution
- Efficient profile lookup indices for O(1) operations
- String builder capacity estimation for reduced allocations

The identified improvements are mostly minor optimizations that would provide marginal performance gains. The code already demonstrates good performance practices overall. The most impactful improvements would be:

1. **Regex compilation optimization** - Move regex compilation to package level for config parsing
2. **Action counting optimization** - Pre-calculate action counts to avoid redundant iterations
3. **String building optimizations** - Eliminate fmt.Sprintf calls in hot paths and use maps for lookups
4. **Key generation optimization** - Use string builders instead of fmt.Sprintf for map key generation

These optimizations would be most beneficial in scenarios with large numbers of profiles or frequent config file parsing operations.