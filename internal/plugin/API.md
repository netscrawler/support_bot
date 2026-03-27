# Plugin System API Documentation

## Table of Contents

1. [Overview](#overview)
2. [Core Interfaces](#core-interfaces)
3. [Manager API](#manager-api)
4. [Plugin Lifecycle](#plugin-lifecycle)
5. [Runtime Configuration](#runtime-configuration)
6. [Data Structures](#data-structures)
7. [Error Handling](#error-handling)
8. [Lua Plugin API](#lua-plugin-api)
9. [Available Modules](#available-modules)
10. [Examples](#examples)

---

## Overview

The plugin system provides a secure, sandboxed environment for executing Lua scripts to extend application functionality without recompilation. It includes:

- Database-backed plugin storage with versioning
- Isolated execution environments (sandboxing)
- Resource controls and security restrictions
- Comprehensive lifecycle management
- JSON data serialization/deserialization

**Architecture:**
```
┌─────────────────────────────────────────┐
│          Application Code               │
└──────────────┬──────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│        Manager (Orchestrator)            │
│  • LoadPluginsFromDBByName()             │
│  • BuildPluginFromDTO()                  │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│      LuaPlugin (Implementation)          │
│  • Init() • Execute() • Validate()       │
│  • Cleanup() • IsHealthy()               │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│       LuaRuntime (Sandbox)               │
│  • Isolated VM • Module whitelist        │
│  • Resource limits • Security            │
└──────────────────────────────────────────┘
```

---

## Core Interfaces

### Plugin Interface

Main interface that all plugins must implement:

```go
type Plugin interface {
    // Metadata methods
    Name() string
    Version() string
    Description() string
    
    // Lifecycle methods
    Init(config map[string]any) error
    Execute(ctx context.Context, params map[string]any) ([]byte, error)
    Validate(params map[string]any) error
    Cleanup() error
    
    // Health check
    IsHealthy() bool
}
```

**Methods:**

- `Name()` - Returns unique plugin identifier
- `Version()` - Returns semantic version string (e.g., "1.2.3")
- `Description()` - Returns human-readable description
- `Init(config)` - Initializes plugin with configuration
- `Execute(ctx, params)` - Main execution function, returns JSON bytes
- `Validate(params)` - Validates parameters before execution
- `Cleanup()` - Releases resources, closes connections
- `IsHealthy()` - Health check for monitoring

### PluginLoader Interface

Interface for loading plugins from storage:

```go
type PluginLoader interface {
    LoadByName(ctx context.Context, name string) (LuaPluginDTO, error)
    LoadByNameAndVersion(ctx context.Context, name, version string) (LuaPluginDTO, error)
    LoadByID(ctx context.Context, id int) (LuaPluginDTO, error)
    LoadByNameAll(ctx context.Context, name string) ([]LuaPluginDTO, error)
}
```

**Methods:**

- `LoadByName(ctx, name)` - Loads latest version of plugin by name
- `LoadByNameAndVersion(ctx, name, version)` - Loads specific version
- `LoadByID(ctx, id)` - Loads plugin by database ID
- `LoadByNameAll(ctx, name)` - Loads all versions of a plugin

---

## Manager API

The Manager orchestrates plugin lifecycle and provides high-level operations.

### Constructor

```go
func NewManager(cfg *Config, repo PluginLoader, std *stdlib.STD) *Manager
```

**Parameters:**
- `cfg` - Plugin system configuration
- `repo` - Plugin repository implementation (database)
- `std` - Standard library for plugins (can be nil)

**Returns:** Configured Manager instance

**Example:**
```go
cfg := &plugins.Config{
    Enable: true,
    AllowedModules: []string{"json", "http"},
}
repo := &PluginRepository{db: database}
manager := plugins.NewManager(cfg, repo, nil)
```

### Loading Plugins

#### LoadPluginsFromDBByName

```go
func (m *Manager) LoadPluginsFromDBByName(
    ctx context.Context,
    name string,
    version *string,
) (*LuaPlugin, error)
```

**Parameters:**
- `ctx` - Context for cancellation/timeout
- `name` - Plugin name
- `version` - Optional version string (nil = latest)

**Returns:**
- `*LuaPlugin` - Loaded and initialized plugin instance
- `error` - Error if loading fails

**Errors:**
- `ErrPluginManagerDisabled` - If plugin system is disabled
- Repository errors (wrapped)
- Lua syntax errors
- Plugin validation errors

**Example:**
```go
// Load latest version
plugin, err := manager.LoadPluginsFromDBByName(ctx, "data_collector", nil)
if err != nil {
    log.Fatal(err)
}

// Load specific version
version := "1.2.3"
plugin, err := manager.LoadPluginsFromDBByName(ctx, "data_collector", &version)
```

#### BuildPluginFromDTO

```go
func (m *Manager) BuildPluginFromDTO(plug LuaPluginDTO) (*LuaPlugin, error)
```

**Parameters:**
- `plug` - Plugin data transfer object with Lua code

**Returns:**
- `*LuaPlugin` - Constructed plugin instance
- `error` - Error if construction fails

**Example:**
```go
dto := LuaPluginDTO{
    Name:      "custom_plugin",
    Version:   "1.0.0",
    PluginStr: luaScriptString,
}
plugin, err := manager.BuildPluginFromDTO(dto)
```

---

## Plugin Lifecycle

### 1. Creation/Loading

```go
// From database
plugin, err := manager.LoadPluginsFromDBByName(ctx, "plugin_name", nil)

// From string
plugin, err := plugins.NewLuaPluginWithConfigFromString(
    scriptString,
    plugins.DefaultRuntimeConfig(),
    nil,
)
```

### 2. Initialization

```go
config := map[string]any{
    "api_key": "secret_key",
    "base_url": "https://api.example.com",
    "timeout": 30,
}
err := plugin.Init(config)
```

### 3. Validation (Optional)

```go
params := map[string]any{
    "endpoint": "/data",
    "filter": "active",
}
if err := plugin.Validate(params); err != nil {
    log.Printf("Invalid parameters: %v", err)
    return
}
```

### 4. Execution

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

data, err := plugin.Execute(ctx, params)
if err != nil {
    log.Fatal(err)
}

// Parse JSON result
var result map[string]any
json.Unmarshal(data, &result)
```

### 5. Cleanup

```go
defer plugin.Cleanup()
```

**Complete Example:**
```go
func usePlugin(manager *Manager) error {
    // Load
    plugin, err := manager.LoadPluginsFromDBByName(ctx, "collector", nil)
    if err != nil {
        return err
    }
    defer plugin.Cleanup()
    
    // Initialize
    if err := plugin.Init(config); err != nil {
        return err
    }
    
    // Validate
    if err := plugin.Validate(params); err != nil {
        return err
    }
    
    // Execute
    data, err := plugin.Execute(ctx, params)
    if err != nil {
        return err
    }
    
    // Process results
    var result map[string]any
    return json.Unmarshal(data, &result)
}
```

---

## Runtime Configuration

### Config Structure

```go
type Config struct {
    Enable           bool          // Enable/disable plugin system
    PluginsDir       string        // Directory for Lua files (deprecated)
    LoadTimeout      time.Duration // Plugin load timeout
    ExecutionTimeout time.Duration // Plugin execution timeout
    MaxMemoryMB      int          // Memory limit per plugin (MB)
    AllowedModules   []string     // Module whitelist
}
```

**Example:**
```go
cfg := &plugins.Config{
    Enable:           true,
    LoadTimeout:      5 * time.Second,
    ExecutionTimeout: 30 * time.Second,
    MaxMemoryMB:      128,
    AllowedModules:   []string{
        "json",
        "http",
        "url",
        "crypto",
        "time",
    },
}
```

### RuntimeConfig Structure

```go
type RuntimeConfig struct {
    AllowedModules []string // Module whitelist
    CallStackSize  int      // Lua call stack size
    RegistrySize   int      // Lua registry size
}
```

**Default Configuration:**
```go
func DefaultRuntimeConfig() *RuntimeConfig {
    return &RuntimeConfig{
        AllowedModules: []string{
            "json", "http", "url",
            "time", "strings", "inspect",
        },
        CallStackSize: 256,
        RegistrySize:  256,
    }
}
```

---

## Data Structures

### LuaPluginDTO

Data transfer object for plugin storage:

```go
type LuaPluginDTO struct {
    Id          int    `db:"id"`
    Name        string `db:"name"`
    Version     string `db:"version"`
    Description string `db:"description"`
    Author      string `db:"author"`
    PluginStr   string `db:"plugin_str"` // Lua code
    CreatedAt   string `db:"created_at"`
    UpdatedAt   string `db:"updated_at"`
}
```

### PluginInfo

Metadata about a plugin:

```go
type PluginInfo struct {
    Name        string  // Plugin name
    Version     string  // Semantic version
    Description string  // Description
    Author      string  // Author name
    FilePath    *string // Optional file path
    DBID        *int    // Database ID
    Healthy     bool    // Health status
}
```

---

## Error Handling

### Predefined Errors

```go
var (
    ErrPluginTableNotFound   = errors.New("plugin table not found")
    ErrPluginNameRequired    = errors.New("plugin name is required")
    ErrPluginInitFailed      = errors.New("plugin init failed")
    ErrValidationFailed      = errors.New("validation failed")
    ErrPluginManagerDisabled = errors.New("plugin manager disabled")
)
```

### Error Checking

```go
plugin, err := manager.LoadPluginsFromDBByName(ctx, "plugin", nil)
if err != nil {
    if errors.Is(err, plugins.ErrPluginManagerDisabled) {
        log.Println("Plugin system is disabled")
        return
    }
    log.Printf("Failed to load plugin: %v", err)
    return
}

if err := plugin.Init(config); err != nil {
    if errors.Is(err, plugins.ErrPluginInitFailed) {
        log.Println("Plugin initialization failed")
    }
    return err
}
```

### Lua Error Handling

Lua errors are automatically captured and wrapped:

```go
data, err := plugin.Execute(ctx, params)
if err != nil {
    // err contains Lua error message
    log.Printf("Lua error: %v", err)
}
```

---

## Lua Plugin API

### Plugin Structure

Every Lua plugin must define a global `plugin` table:

```lua
plugin = {
    -- Metadata (required)
    name = "plugin_name",
    version = "1.0.0",
    description = "Plugin description",
    author = "Author Name",
    
    -- Lifecycle functions (all required)
    init = function(config) ... end,
    execute = function(params) ... end,
    validate = function(params) ... end,
    cleanup = function() ... end
}
```

### init Function

**Signature:** `function(config) -> (success: bool, error: string|nil)`

Called once when plugin is loaded. Use to initialize state.

```lua
init = function(config)
    -- Save configuration
    plugin.api_key = config.api_key
    plugin.base_url = config.base_url
    
    -- Validate required config
    if not plugin.api_key then
        return false, "api_key is required"
    end
    
    return true, nil
end
```

### execute Function

**Signature:** `function(params) -> (data: table|string, error: string|nil)`

Main execution function. Return data and optional error.

```lua
execute = function(params)
    local http = require("http")
    local json = require("json")
    
    -- Make HTTP request
    local url = plugin.base_url .. params.endpoint
    local response, err = http.request("GET", url)
    if err then
        return nil, "Request failed: " .. err
    end
    
    -- Parse response
    local data = json.decode(response.body)
    return data, nil
end
```

### validate Function

**Signature:** `function(params) -> (valid: bool, error: string|nil)`

Validates parameters before execution.

```lua
validate = function(params)
    if not params.endpoint then
        return false, "endpoint is required"
    end
    
    if not params.method or (params.method ~= "GET" and params.method ~= "POST") then
        return false, "method must be GET or POST"
    end
    
    return true, nil
end
```

### cleanup Function

**Signature:** `function() -> ()`

Cleanup resources when plugin is unloaded.

```lua
cleanup = function()
    -- Clear sensitive data
    plugin.api_key = nil
    plugin.config = nil
end
```

---

## Available Modules

### json (gopher-json)

JSON encoding/decoding:

```lua
local json = require("json")

-- Encode
local str = json.encode({key = "value"})

-- Decode
local data = json.decode('{"key":"value"}')
```

### http (gluahttp)

HTTP client:

```lua
local http = require("http")

-- GET request
local response, err = http.request("GET", "https://api.example.com/data")
if err then
    return nil, err
end

-- POST request with headers
local response, err = http.request("POST", url, {
    headers = {
        ["Content-Type"] = "application/json",
        ["Authorization"] = "Bearer " .. token
    },
    body = json.encode({data = "value"})
})

-- Response fields
print(response.status_code)  -- 200
print(response.body)         -- Response body string
```

### url (gluaurl)

URL parsing and building:

```lua
local url = require("url")

-- Parse URL
local parsed = url.parse("https://example.com/path?key=value")
print(parsed.scheme)  -- https
print(parsed.host)    -- example.com
print(parsed.path)    -- /path

-- Build URL
local built = url.build({
    scheme = "https",
    host = "api.example.com",
    path = "/v1/data",
    query = "filter=active"
})
```

### crypto (gluacrypto)

Cryptographic functions:

```lua
local crypto = require("crypto")

-- MD5
local hash = crypto.md5("text")

-- SHA256
local hash = crypto.sha256("text")

-- HMAC
local signature = crypto.hmac("sha256", "secret", "message")
```

### Standard Lua Modules

Always available:

- `table` - Table operations (insert, remove, sort, concat)
- `string` - String functions (upper, lower, sub, gsub, format)
- `math` - Mathematical functions (abs, ceil, floor, max, min, random)
- `os.time()`, `os.clock()`, `os.date()` - Time functions only

**Disabled for security:**
- `io.*` - File operations
- `os.execute`, `os.remove`, etc. - System operations
- `debug.*` - Debug module
- `dofile`, `loadfile` - File loading

---

## Examples

### HTTP Data Collector

```lua
plugin = {
    name = "http_collector",
    version = "1.0.0",
    description = "Collects data from HTTP API",
    author = "DevTeam",
    
    init = function(config)
        plugin.base_url = config.base_url
        plugin.api_key = config.api_key
        plugin.timeout = config.timeout or 30
        
        if not plugin.api_key then
            return false, "api_key is required"
        end
        
        return true, nil
    end,
    
    execute = function(params)
        local http = require("http")
        local json = require("json")
        
        -- Build URL
        local url = plugin.base_url .. params.endpoint
        
        -- Prepare headers
        local headers = {
            ["Authorization"] = "Bearer " .. plugin.api_key,
            ["Content-Type"] = "application/json",
            ["Accept"] = "application/json"
        }
        
        -- Make request
        local response, err = http.request("GET", url, {
            headers = headers,
            timeout = plugin.timeout
        })
        
        if err then
            return nil, "HTTP request failed: " .. err
        end
        
        -- Check status
        if response.status_code ~= 200 then
            return nil, "Bad status code: " .. response.status_code
        end
        
        -- Parse JSON
        local data = json.decode(response.body)
        
        -- Add metadata
        return {
            data = data,
            collected_at = os.time(),
            source = params.endpoint
        }, nil
    end,
    
    validate = function(params)
        if not params.endpoint then
            return false, "endpoint is required"
        end
        return true, nil
    end,
    
    cleanup = function()
        plugin.api_key = nil
    end
}
```

### Data Transformation Plugin

```lua
plugin = {
    name = "data_enricher",
    version = "1.0.0",
    description = "Enriches data with additional fields",
    author = "DataTeam",
    
    init = function(config)
        plugin.config = config
        return true, nil
    end,
    
    execute = function(params)
        local result = {}
        
        -- Process each dataset
        for key, items in pairs(params.datasets) do
            result[key] = {}
            
            -- Enrich each item
            for i, item in ipairs(items) do
                local enriched = {
                    -- Original fields
                    id = item.id,
                    name = item.name,
                    value = item.value,
                    
                    -- New fields
                    processed = true,
                    processed_at = os.time(),
                    index = i,
                    dataset = key,
                    
                    -- Calculated fields
                    value_doubled = (item.value or 0) * 2,
                    name_upper = string.upper(item.name or "")
                }
                
                table.insert(result[key], enriched)
            end
        end
        
        return result, nil
    end,
    
    validate = function(params)
        if not params.datasets then
            return false, "datasets is required"
        end
        
        if type(params.datasets) ~= "table" then
            return false, "datasets must be a table"
        end
        
        return true, nil
    end,
    
    cleanup = function()
        plugin.config = nil
    end
}
```

### Multi-Source Aggregator

```lua
plugin = {
    name = "multi_source_aggregator",
    version = "2.0.0",
    description = "Aggregates data from multiple sources",
    author = "IntegrationTeam",
    
    init = function(config)
        plugin.sources = config.sources or {}
        
        if #plugin.sources == 0 then
            return false, "at least one source is required"
        end
        
        return true, nil
    end,
    
    execute = function(params)
        local http = require("http")
        local json = require("json")
        local results = {}
        
        -- Fetch from each source
        for i, source in ipairs(plugin.sources) do
            local url = source.url .. (params.path or "")
            local response, err = http.request("GET", url, {
                headers = {["Authorization"] = source.token}
            })
            
            if response and response.status_code == 200 then
                local data = json.decode(response.body)
                table.insert(results, {
                    source = source.name,
                    data = data,
                    fetched_at = os.time()
                })
            else
                table.insert(results, {
                    source = source.name,
                    error = err or "status: " .. response.status_code
                })
            end
        end
        
        -- Aggregate results
        return {
            total_sources = #plugin.sources,
            successful = countSuccessful(results),
            results = results
        }, nil
    end,
    
    validate = function(params)
        return true, nil
    end,
    
    cleanup = function()
        plugin.sources = nil
    end
}

-- Helper function
function countSuccessful(results)
    local count = 0
    for _, result in ipairs(results) do
        if result.data then
            count = count + 1
        end
    end
    return count
end
```

---

## Best Practices

1. **Always handle errors** - Check for nil returns and error strings
2. **Validate inputs** - Use validate() function comprehensively
3. **Use timeouts** - Set reasonable timeouts for HTTP requests
4. **Clean up resources** - Implement cleanup() properly
5. **Avoid global state** - Store state in plugin table
6. **Test thoroughly** - Test with various inputs and edge cases
7. **Document your plugin** - Include clear descriptions and examples
8. **Version semantically** - Follow semver (major.minor.patch)
9. **Handle rate limits** - Implement backoff for external APIs
10. **Log appropriately** - Return meaningful error messages

---

## Security Considerations

1. **Never trust input** - Always validate and sanitize parameters
2. **Protect credentials** - Clear sensitive data in cleanup()
3. **Limit external access** - Only enable required modules
4. **Validate URLs** - Check for SSRF vulnerabilities
5. **Set timeouts** - Prevent infinite loops and hangs
6. **Monitor execution** - Track plugin performance and errors
7. **Review code** - Audit plugins before production deployment
8. **Use least privilege** - Minimize allowed modules
9. **Sandbox violations** - Report attempts to access forbidden APIs
10. **Update dependencies** - Keep Lua VM and modules updated

---

## Version History

- **2.0.0** (March 2026)
  - Database-backed plugin storage
  - Version management
  - Improved sandbox security
  - stdlib support
  
- **1.0.0** (Initial Release)
  - Basic plugin system
  - File-based plugins
  - Lua sandbox
  - Core modules

---

## Support

For issues, questions, or contributions:
- Check existing tests for examples
- Review error messages carefully
- Consult Lua documentation
- Test in isolation before deploying

---

*Last Updated: March 6, 2026*
