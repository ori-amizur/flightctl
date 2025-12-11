---
project_name: 'flightctl'
user_name: 'Ori'
date: '2025-12-09T12:36:40+02:00'
sections_completed:
  ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'quality_rules', 'workflow_rules', 'anti_patterns']
status: 'complete'
rule_count: 45
optimized_for_llm: true
---

# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## Technology Stack & Versions

**Core Technologies:**
- **Go:** 1.24.0 (toolchain: go1.24.6)
- **Testing:** `testify/require`, `gomock` (go.uber.org/mock)
- **Logging:** `logrus` v1.9.3, internal `pkg/log` package
- **Container:** `containers/image/v5` v5.30.1
- **Utilities:** `samber/lo` v1.49.1

**Key Dependencies:**
- API types: `github.com/flightctl/flightctl/api/v1beta1` (internal)
- Error handling: `fmt` standard library (error wrapping)
- Context: `context` standard library

## Critical Implementation Rules

### Language-Specific Rules (Go)

**Error Handling:**
- Always wrap errors: `fmt.Errorf("context: %w", err)`
- Include context in error messages (what operation, which resource)
- Use predefined errors from `agent/device/errors` package when applicable
- Never return errors that block reconciliation (log and continue for pruning operations)

**Context Usage:**
- Context must be first parameter: `func Method(ctx context.Context, ...) error`
- Pass context to all external calls (Podman, file I/O, spec reading)
- Respect context cancellation in long-running operations

**Interface Verification:**
- Always verify interface compliance: `var _ Manager = (*manager)(nil)`
- Place immediately after type definitions

**Manager Pattern:**
- Public interface: `type Manager interface { ... }`
- Private struct: `type manager struct { ... }`
- Constructor: `func NewManager(...) Manager`
- Dependency injection via constructor parameters

**File I/O:**
- ALL disk operations MUST use `fileio.ReadWriter` interface
- Never use `os` package directly for file operations
- This enables testing and simulation

**Dependency Management:**
- Minimal dependencies principle: "A little copying is better than a little dependency"
- Strongly vet new dependencies - verify existing code first
- Serial operations preferred over parallel (resource conservation > speed)

### Framework-Specific Rules

**Agent Architecture:**
- Managers in `internal/agent/device/<namespace>/` (mirrors spec keys)
- One manager per package
- Interface and implementation in same file (`manager.go`)
- Tests co-located (`manager_test.go`)

**Threading & Lifecycle:**
- Primarily single-threaded (async requires strong justification)
- Serial operations preferred over parallel
- Graceful termination via `agent/shutdown` package
- Configuration reload via `agent/reload` package

**Extending Functionality:**
- Use functional options pattern: `func New(opts ...Option)`
- "One way to do things" - no duplicate functionality
- Integrate with existing patterns

### Testing Rules

**Test Structure:**
- Use `testify/require`: `require := require.New(t)`
- Table-driven tests with inline `setupMocks` functions
- Test file naming: `*_test.go` in same package

**Mock Usage:**
- Use `gomock` for mocking dependencies
- Always use `defer ctrl.Finish()` after creating gomock controller
- Mock external dependencies: Podman client, spec manager, file I/O
- Use gomock expectations for behavior verification

**Test Organization:**
- Unit tests: `manager_test.go` in same package as implementation
- Integration tests: `device_test.go` for agent lifecycle integration
- Test both success and failure paths
- Test edge cases: empty lists, concurrent operations, partial failures

**Test Example Pattern:**
```go
func TestManager_Prune(t *testing.T) {
    require := require.New(t)
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    testCases := []struct {
        name        string
        setupMocks  func(*client.MockPodman, *spec.MockManager)
        wantError   bool
    }{
        // Test cases...
    }
    // Test execution...
}
```

### Code Quality & Style Rules

**Naming Conventions:**
- Package names: lowercase, single word (e.g., `pruning`, `applications`)
- Manager interface: `Manager` (public, capitalized)
- Manager struct: `manager` (private, lowercase)
- Constructor: `NewManager(...) Manager`
- Methods: PascalCase for public, camelCase for private helpers

**Code Organization:**
- Managers in `internal/agent/device/<namespace>/` (mirrors spec keys)
- One manager per package
- Interface and implementation in same file (`manager.go`)
- Tests co-located (`manager_test.go`)

### Development Workflow Rules

**PR Requirements:**
- Concise and minimal changes
- Demonstrate understanding of agent internals
- No unnecessary dependencies
- Uses fileio for ALL disk operations
- Spec access via spec manager only
- No unwarranted async code
- PR is minimal and focused
- One way to do things

**Code Review Checklist:**
- [ ] No unnecessary dependencies
- [ ] Uses fileio for ALL disk operations
- [ ] Spec access via spec manager only
- [ ] No unwarranted async code
- [ ] PR is minimal and focused
- [ ] One way to do things

### Critical Don't-Miss Rules

**Anti-Patterns to Avoid:**
- ❌ DON'T: Use `os` package directly for file operations (use `fileio.ReadWriter`)
- ❌ DON'T: Block reconciliation on errors (log and continue for pruning)
- ❌ DON'T: Skip context parameter (always first parameter)
- ❌ DON'T: Use global state (dependency injection only)
- ❌ DON'T: Ignore errors silently (always log errors)
- ❌ DON'T: Add unnecessary dependencies (verify existing code first)
- ❌ DON'T: Create async code without strong justification
- ❌ DON'T: Access spec files directly (use spec manager)

**Edge Cases:**
- Handle empty image lists gracefully
- Handle partial failures (some images fail to remove)
- Handle concurrent spec updates
- Handle Podman API failures
- Validate rollback capability after pruning

**Security Rules:**
- Never prune OS images (bootc manages these)
- Always verify required images exist before pruning
- Never prune images referenced in current or rollback specs

**Performance Gotchas:**
- Pruning must not block reconciliation
- Serial operations preferred (resource conservation > speed)
- Stateless operations (determine eligibility on each run)

---

## Usage Guidelines

**For AI Agents:**

- Read this file before implementing any code
- Follow ALL rules exactly as documented
- When in doubt, prefer the more restrictive option
- Update this file if new patterns emerge

**For Humans:**

- Keep this file lean and focused on agent needs
- Update when technology stack changes
- Review quarterly for outdated rules
- Remove rules that become obvious over time

**Last Updated:** 2025-12-09T12:36:40+02:00

