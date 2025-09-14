# Middleware Architecture Design

## Overview

This document outlines the planned middleware system for Conduit using gorilla/mux with directory-based middleware organization. This approach provides intuitive, scalable middleware management that aligns perfectly with our existing directory-based route discovery system.

## Architecture Concept

### Directory-Based Middleware Inheritance

```
/
├── middleware.go              # Global middleware (applied to ALL routes)
├── api/
│   ├── middleware.go          # Applied to all /api/* routes
│   └── v1/
│       ├── middleware.go      # Applied to all /api/v1/* routes
│       ├── users/
│       │   ├── route.go       # GET, POST handlers
│       │   ├── middleware.go  # Applied only to /api/v1/users/* routes
│       │   └── id_/
│       │       ├── route.go   # GET, DELETE handlers for /api/v1/users/{id}
│       │       └── middleware.go  # Applied only to /api/v1/users/{id}/* routes
│       └── profiles/
│           ├── route.go
│           └── id_/
│               ├── route.go
│               └── middleware.go
```

### Middleware Application Order

Middleware is applied in directory hierarchy order (most global to most specific):

1. **Global**: `/middleware.go`
2. **API Level**: `/api/middleware.go`
3. **Version Level**: `/api/v1/middleware.go`
4. **Resource Level**: `/api/v1/users/middleware.go`
5. **Route Handler**: `/api/v1/users/route.go -> GET()`

## Example Implementation

### Sample Middleware File

```go
// /api/v1/middleware.go
package v1

import (
    "net/http"
    "time"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // JWT validation logic
        token := r.Header.Get("Authorization")
        if !isValidToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if isRateLimited(r.RemoteAddr) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Generated Route with Middleware

```go
// Generated: .conduit/go/routes/api/v1/users/gen_route.go
package users_gen

import (
    "net/http"
    "github.com/gorilla/mux"

    // Import original handlers
    users "my-app/api/v1/users"

    // Import middleware from hierarchy
    globalMW "my-app"
    apiMW "my-app/api"
    v1MW "my-app/api/v1"
    usersMW "my-app/api/v1/users"
)

// SetupRoutes configures the router with middleware chain
func SetupRoutes() *mux.Router {
    r := mux.NewRouter()

    // Apply middleware in hierarchy order
    r.Use(globalMW.LoggingMiddleware)     // Global
    r.Use(apiMW.CORSMiddleware)           // API level
    r.Use(v1MW.AuthMiddleware)            // Version level
    r.Use(v1MW.RateLimitMiddleware)       // Version level
    r.Use(usersMW.ValidationMiddleware)   // Resource level

    // Register handlers
    r.HandleFunc("/api/v1/users", users.GET).Methods("GET")
    r.HandleFunc("/api/v1/users", users.POST).Methods("POST")

    return r
}
```

## Implementation Phases

### Phase 1: Gorilla/Mux Migration
- **Goal**: Replace `http.ServeMux` with `gorilla/mux`
- **Tasks**:
  - Add gorilla/mux dependency
  - Update route registration templates
  - Change path patterns (`/users/:id` → `/users/{id}`)
  - Update registry to use mux routers

### Phase 2: Middleware Detection & Parsing
- **Goal**: Extend route walker to discover middleware files
- **Tasks**:
  - Enhance `RouteWalker` to detect `middleware.go` files
  - Create AST parsing for middleware functions
  - Build middleware hierarchy during tree construction
  - Add middleware info to route models

### Phase 3: Code Generation Integration
- **Goal**: Generate middleware application in route files
- **Tasks**:
  - Create middleware templates for generated code
  - Implement middleware import management
  - Generate `SetupRoutes()` with middleware chains
  - Update registry to compose middleware-enabled routers

### Phase 4: Advanced Features
- **Goal**: Add sophisticated middleware features
- **Tasks**:
  - Conditional middleware application
  - Middleware configuration support
  - Error handling and recovery middleware
  - Performance monitoring integration

## Pros & Cons Analysis

### Advantages

1. **Intuitive Organization**
   - Middleware scope matches directory structure
   - Easy to understand inheritance hierarchy
   - Natural place to add route-specific middleware

2. **Perfect Cache Integration**
   - Changing middleware.go only regenerates affected routes
   - File-level cache invalidation works seamlessly
   - Performance scales with our caching strategy

3. **Gorilla/Mux Benefits**
   - Built-in middleware chaining with `r.Use()`
   - Better path parameter handling: `/users/{id}`
   - Subrouter support for clean organization
   - Mature, well-tested library

4. **Scalable Architecture**
   - No central configuration files to maintain
   - Middleware grows naturally with route structure
   - Easy to add/remove without global changes

### Disadvantages

1. **Additional Dependency**
   - Gorilla/mux adds external dependency
   - Slightly larger binary size
   - Migration effort from stdlib

2. **Complexity Increase**
   - More complex template generation
   - Import management becomes trickier
   - Middleware order reasoning required

3. **Learning Curve**
   - Developers need to understand inheritance rules
   - More files to maintain per route

## Migration Strategy

### Backward Compatibility
- Implement alongside existing system initially
- Provide configuration flag to enable new system
- Gradual migration path for existing projects

### Testing Strategy
- Unit tests for middleware detection
- Integration tests for middleware application
- Performance benchmarks vs current system

## Future Considerations

### Configuration Options
```yaml
# conduit.yaml
middleware:
  discovery: true          # Enable middleware.go file discovery
  strict_hierarchy: true   # Enforce directory-based inheritance
  global_middleware: []    # Explicit global middleware list
```

### Advanced Features
- Middleware annotations in comments
- Conditional middleware application
- Middleware dependency injection
- Performance monitoring hooks

## Implementation Timeline

This feature will be implemented **after** completing the current full code generation system. The middleware architecture builds upon and enhances our per-route generation approach, making it a natural next evolution of the system.

---

*This document serves as a reference for future implementation. All design decisions and technical approaches documented here should be reviewed before implementation begins.*