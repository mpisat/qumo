# Go Test Code Generation Rules

This prompt defines the rules and patterns to follow when generating test code for Go projects.

## Core Principles

### 1. Test Function Naming Conventions
- **Basic Tests**: `Test[FunctionName]`
- **Method Tests**: `Test[StructName]_[MethodName]`
- **Method Tests with Options**: `Test[StructName]_[MethodName]_[Option]`

Examples:
`8. **Mock Package Placement**: All mocks must be in the same package (`package [packagename]`), never in `_test` packages
9. **Mock File Organization**: Use dedicated `mock_[feature]_test.go` files for mock definitions
10. **Mock Structure Flexibility**: Choose between `mock.Mock` embedding and function fields based on the complexity of the interface and testing needso
```go
func TestNewFrame(t *testing.T) { /* Basic test */ }
func TestGroupSequence_String(t *testing.T) { /* Method test */ }
func TestParameters_String_NilValue(t *testing.T) { /* Method test with option */ }
```

### 2. Test Case Structure
**Mandatory**: Test cases must use `map[string]struct{...}` format with test case names as keys for table-driven tests **only when there are multiple cases**.

For single test cases, write the test code directly without using a map.

```go
// Single test case example
func TestExample_SingleCase(t *testing.T) {
    input := "test"
    expected := "test_result"

    result, err := SomeFunction(input)

    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}

// Multiple test cases example
func TestExample_MultipleCases(t *testing.T) {
    tests := map[string]struct {
        input    string
        expected string
        wantErr  bool
    }{
        "valid input": {
            input:    "test",
            expected: "test_result",
            wantErr:  false,
        },
        "empty input": {
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            result, err := SomeFunction(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### 3. Required Packages and Assertions
- **Use testify package**: All tests must use `github.com/stretchr/testify`
- **assert for comparisons**: `assert.Equal(t, expected, actual)`, `assert.NoError(t, err)`, etc.
- **require for preconditions**: `require.NoError(t, err)`, `require.NotNil(t, obj)`, etc.
- **mock for mocking**: Use `github.com/stretchr/testify/mock`

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)
```

### 4. Mock Definitions and Management
**Mandatory**: Mock structures must be defined in the same package (`package [packagename]`) within dedicated `mock_XXX_test.go` files.

#### Mock File Organization
- **Existing Mocks**: Check for existing mock files before creating new ones
- **File Naming**: `mock_[feature]_test.go` (e.g., `mock_group_reader_test.go`, `mock_track_writer_test.go`)
- **Package Declaration**: Always use `package [packagename]` (not `package [packagename]_test`)

#### Mock Structure Pattern
Mock structures can be implemented using either of the following patterns:

**Pattern 1: Using testify/mock** (Recommended for complex mocking):
```go
// In mock_example_test.go
package moqt

import (
    "github.com/stretchr/testify/mock"
)

var _ SomeInterface = (*MockExample)(nil)

type MockExample struct {
    mock.Mock
}

func (m *MockExample) SomeMethod(arg string) error {
    args := m.Called(arg)
    return args.Error(0)
}
```

**Pattern 2: Using function fields** (Suitable for simpler interfaces):
```go
// In mock_example_test.go
package moqt

var _ SomeInterface = (*MockExample)(nil)

type MockExample struct {
    SomeMethodFunc func(arg string) error
}

func (m *MockExample) SomeMethod(arg string) error {
    if m.SomeMethodFunc != nil {
        return m.SomeMethodFunc(arg)
    }
    return nil
}
```

#### Mock Implementation Rules
1. **Implementation Options**: Mock structs can either embed `mock.Mock` for complex mocking scenarios or use function fields for simpler cases
2. **Testify/Mock Pattern**: When using `mock.Mock`, all mock methods MUST use `m.Called()` for argument capture and return value handling
3. **Function Field Pattern**: When using function fields, check if the function is not nil before calling it
4. **Expectation Verification**: When using testify/mock, always call `mockObj.AssertExpectations(t)` in tests

#### Mock Discovery Rules
1. **Search First**: Before creating new mocks, search for existing mock files in the same package
2. **Reuse Existing**: If a mock already exists, use it instead of creating a duplicate
3. **Create New**: Only create new mock files when no suitable mock exists
4. **Consistent Naming**: Follow the existing pattern: `Mock[InterfaceName]` (e.g., `MockGroupReader`, `MockTrackWriter`)
5. **No Factory Functions**: Do not create `createMock...()`, `newMock...()`, or similar helper functions for mock initialization

### 5. Package Declaration
- **Internal Tests**: `package [packagename]` - Can test private functions and access internal structures
- **External Tests**: `package [packagename]_test` - Test only public APIs
- **Mock Files**: Always use `package [packagename]` - Mocks must be in the same package as the interfaces they implement

### 6. Interface Testing Policy
**PROHIBITED**: Do not create test files or test functions for interface definitions themselves.

#### Rules for Interface Files
1. **No Test File Generation**: Never create `[interface_name]_test.go` files for files that only contain interface definitions
2. **No Interface Test Functions**: Do not write test functions that test interface behavior directly
3. **Interface Testing Through Implementations**: Test interfaces indirectly through their concrete implementations
4. **Mock Creation Only**: For interface files, only create corresponding mock files in `mock_[interface_name]_test.go`

#### Examples of What NOT to Do
```go
// ❌ PROHIBITED: Do not create test files for interface-only files
// File: writer_interface_test.go
func TestWriterInterface(t *testing.T) {
    // This should never exist
}

// ❌ PROHIBITED: Do not test interface methods directly
func TestAnnouncementWriter_SendAnnouncements(t *testing.T) {
    // Interface methods should not be tested directly
}
```

#### Correct Approach
```go
// ✅ CORRECT: Test concrete implementations that implement the interface
func TestConcreteWriter_SendAnnouncements(t *testing.T) {
    writer := NewConcreteWriter()
    // Test the concrete implementation
}

// ✅ CORRECT: Create mocks for interfaces in mock files
// File: mock_announcement_writer_test.go
type MockAnnouncementWriter struct {
    mock.Mock
}
```

## Test File Structure

### File Naming
- `[filename]_test.go` (regular test files)
- `mock_[feature]_test.go` (dedicated mock files, always in same package)
- **PROHIBITED**: Do not create test files for interface-only source files

### Test Categories

#### 1. Unit Tests
Test basic functionality of each feature:
```go
func TestNewObject(t *testing.T) {
    obj := NewObject("param")
    assert.NotNil(t, obj)
}
```

#### 2. Method Tests
Test struct methods with comprehensive cases:
```go
func TestObject_Method(t *testing.T) {
    tests := map[string]struct {
        setup    func() *Object
        input    string
        expected string
        wantErr  bool
    }{
        "success case": {
            setup: func() *Object { return NewObject("test") },
            input: "input",
            expected: "expected",
            wantErr: false,
        },
        "error case": {
            setup: func() *Object { return NewObject("") },
            input: "invalid",
            expected: "",
            wantErr: true,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            obj := tt.setup()
            result, err := obj.Method(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

#### 3. Error Handling Tests
Explicitly test error cases:
```go
func TestObject_Method_Error(t *testing.T) {
    tests := map[string]struct {
        setup       func() *Object
        input       string
        expectError bool
        errorType   error
    }{
        "invalid input": {
            setup:       func() *Object { return NewObject("") },
            input:       "",
            expectError: true,
            errorType:   ErrInvalidInput,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            obj := tt.setup()
            _, err := obj.Method(tt.input)
            
            if tt.expectError {
                assert.Error(t, err)
                if tt.errorType != nil {
                    assert.ErrorIs(t, err, tt.errorType)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 4. Boundary Value Tests
Test edge cases, minimum, maximum, zero values:
```go
func TestObject_BoundaryValues(t *testing.T) {
    tests := map[string]struct {
        value    int
        expected bool
    }{
        "zero value":     {value: 0, expected: false},
        "minimum value":  {value: 1, expected: true},
        "maximum value":  {value: math.MaxInt, expected: true},
        "negative value": {value: -1, expected: false},
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            result := ValidateValue(tt.value)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### 5. Helper Functions
Define test helper functions in the same file or dedicated `*_test.go` files:

```go
// createTestContext creates a test context for testing purposes
func createTestContext() *Context {
    return &Context{
        // Test configuration
    }
}

// Helper function to verify heap property
func isValidHeap(h *Heap) bool {
    // Helper logic
    return true
}
```

#### 6. Concurrent Testing
Use proper synchronization for concurrent tests. Prefer `testing/synctest` for tests that depend on goroutine scheduling, timers (time.Sleep, time.Timer, time.Ticker), or deterministic ordering of concurrent operations. `synctest` lets you run a test in an isolated "bubble" where time is deterministic and goroutine lifetimes are enforced.

The basic rule: If your test spawns goroutines that must be observed deterministically, rely on `synctest.Test` + `synctest.Wait` instead of ad-hoc sleeps or global waits.

```go
import (
    "testing"
    "testing/synctest"
    "sync"
)

func TestConcurrentAccess(t *testing.T) {
    // Simple sync.WaitGroup case — local WaitGroup associated with the bubble
    synctest.Test(t, func(t *testing.T) {
        var wg sync.WaitGroup
        wg.Add(1)
        go func() {
            defer wg.Done()
            // concurrent work
        }()
        // Wait deterministically for all goroutines to block or finish
        wg.Wait()
    })
}
```

Notes and rules for `testing/synctest`:
- Use `synctest.Test(t, func(t *testing.T) { ... })` to run tests in an isolated bubble.
- Use `synctest.Wait()` to wait until all goroutines in the bubble (other than the caller) are durably blocked. This is preferred to `time.Sleep` in tests that rely on scheduling or timers.
- `Wait` must not be called outside a synctest bubble and must not be called concurrently by multiple goroutines in the same bubble.
- Do not call `t.Run`, `t.Parallel`, or `t.Deadline` from within the synctest bubble; they are not supported.
- `T.Cleanup` functions registered within the bubble run inside the bubble and execute immediately before the bubble exits.
 - `T.Cleanup` functions registered within the bubble run inside the bubble and execute immediately before the bubble exits.
 - `T.Context()` returns a `context.Context` with a `Done` channel associated with the bubble; timeouts and cancellations created from that context are scoped to the bubble.
- Local `sync.WaitGroup`s (e.g., `var wg sync.WaitGroup`) become associated with the current bubble when `Add` or `Go` is called from that bubble. Do not rely on package-level `var wg sync.WaitGroup` variables — they cannot be associated with a bubble and may not be durably blocking.
- Some operations are not considered durably blocking and therefore will not cause `synctest.Wait()` to return, such as `sync.Mutex` lock waits, network I/O, or system calls. Prefer in-process fakes (e.g., `net.Pipe`) or mocks when testing network behavior.
- Operations that are durably blocking include blocking `chan` send/receive (for channels created within the bubble), `sync.Cond.Wait`, `sync.WaitGroup.Wait` (when `Add`/`Go` was called in the bubble), and `time.Sleep`.
- Timers and time-dependent code: In a synctest bubble, time advances only when all goroutines are durably blocked; this makes testing time-related behavior deterministic.

Examples:

```go
// Deterministic time example: time in the bubble starts at midnight UTC 2000-01-01
func TestTimeAdvance(t *testing.T) {
    // imports for this snippet: testing, testing/synctest, time
    synctest.Test(t, func(t *testing.T) {
        start := time.Now()
        go func() {
            time.Sleep(1 * time.Second)
            t.Log(time.Since(start)) // always logs "1s"
        }()
        // This later sleep returns after the goroutine above progressed
        time.Sleep(2 * time.Second)
        t.Log(time.Since(start))    // always logs "2s"
    })
}

// Using Wait to observe completion/state
func TestWait(t *testing.T) {
    // imports for this snippet: testing, testing/synctest, github.com/stretchr/testify/assert
    synctest.Test(t, func(t *testing.T) {
        done := false
        go func() {
            done = true
        }()
        synctest.Wait()
        assert.True(t, done)
    })
}
```

When to prefer synctest:
- Tests that rely on timers, `time.Sleep`, `time.Timer`, or `time.Ticker`.
- Tests that start background goroutines where you want deterministic lifecycle management.
- Tests that assert interactions that depend on a specific order or blocking behavior.

When not to use synctest:
- Tests that require real network I/O or external processes; prefer mocking or in-process fakes.
- Tests that must use `t.Parallel` or nested `t.Run` with subtests using parallelism.

Other concurrent testing best practices:
- Keep goroutines deterministic where possible by using `synctest` or explicit synchronization. Avoid arbitrary `time.Sleep` calls.
- Ensure tests clean up all goroutines by the end of the test; `synctest.Test` will panic on deadlock.
- Document any non-obvious synchronization in test code comments to avoid flaky tests.

#### 7. Context and Timeout Tests
Tests involving context or timeouts:
```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    // Test implementation
}
```

## Assertion Patterns

### Basic Comparisons
```go
assert.Equal(t, expected, actual)
assert.NotEqual(t, notExpected, actual)
assert.True(t, condition)
assert.False(t, condition)
assert.Nil(t, object)
assert.NotNil(t, object)
```

### Error Handling
```go
assert.NoError(t, err)
assert.Error(t, err)
assert.ErrorIs(t, err, expectedErr)
assert.ErrorAs(t, err, &target)
```

### Collections
```go
assert.Len(t, collection, expectedLength)
assert.Empty(t, collection)
assert.Contains(t, collection, element)
assert.ElementsMatch(t, expected, actual)
```

## Best Practices

1. **Independent Test Cases**: Do not share state between test cases
2. **Clear Test Names**: Test case names should clearly indicate what is being tested
3. **Boundary Value Testing**: Test zero values, maximum values, minimum values, nil values
4. **Error Cases**: Test both normal and error cases
5. **Mock Discovery**: Always search for existing mocks before creating new ones
6. **Mock Package Placement**: All mocks must be in the same package (`package [packagename]`), never in `_test` packages
7. **Mock File Organization**: Use dedicated `mock_[feature]_test.go` files for mock definitions
8. **Mock Structure Requirements**: **MANDATORY** - All mock structures MUST embed `mock.Mock` from testify/mock
9. **Mock Initialization**: Initialize mocks directly in each test function - do not create factory functions or helper functions for mock initialization
10. **Interface Testing Policy**: **PROHIBITED** - Never create test files or test functions for interface definitions themselves
11. **Setup and Cleanup**: Use setup/teardown functions when necessary
12. **Comments**: Add comments for complex test logic

## Mock Patterns and Management

### Mock File Discovery
Before creating any mock, always search for existing mock implementations:
1. Look for `mock_[feature]_test.go` files in the same package
2. Check if the required mock interface already exists
3. Only create new mock files when no suitable mock exists

### Mock File Creation
When creating new mock files:
```go
// File: mock_example_test.go
package moqt  // Always same package, never _test

// Option 1: Using testify/mock
import (
    "github.com/stretchr/testify/mock"
)

var _ ExampleInterface = (*MockExample)(nil)

type MockExample struct {
    mock.Mock
}

func (m *MockExample) SomeMethod(arg string) error {
    args := m.Called(arg)
    return args.Error(0)
}

// Option 2: Using function fields
var _ ExampleInterface = (*MockExampleFunc)(nil)

type MockExampleFunc struct {
    SomeMethodFunc func(arg string) error
}

func (m *MockExampleFunc) SomeMethod(arg string) error {
    if m.SomeMethodFunc != nil {
        return m.SomeMethodFunc(arg)
    }
    return nil
}
```

### Mock Initialization Rules
Mock structures can be initialized directly within each test function using either pattern.

#### Mock Initialization Examples
```go
func TestWithMock(t *testing.T) {
    // Using testify/mock pattern
    mockObj := &MockExample{}
    mockObj.On("SomeMethod", "input").Return(nil)
    
    err := mockObj.SomeMethod("input")
    
    assert.NoError(t, err)
    mockObj.AssertExpectations(t)
}

func TestWithFunctionFieldMock(t *testing.T) {
    // Using function field pattern
    mockObj := &MockExampleFunc{
        SomeMethodFunc: func(arg string) error {
            assert.Equal(t, "input", arg)
            return nil
        },
    }
    
    err := mockObj.SomeMethod("input")
    assert.NoError(t, err)
}

func TestTableDrivenWithMocks(t *testing.T) {
    tests := map[string]struct {
        setupMock func() ExampleInterface
        input     string
        expected  string
        wantErr   bool
    }{
        "success case with testify/mock": {
            setupMock: func() ExampleInterface {
                mockObj := &MockExample{}
                mockObj.On("SomeMethod", "input").Return("result", nil)
                return mockObj
            },
            input:    "input",
            expected: "result",
            wantErr:  false,
        },
        "success case with function field": {
            setupMock: func() ExampleInterface {
                return &MockExampleFunc{
                    SomeMethodFunc: func(arg string) error {
                        return nil
                    },
                }
            },
            input:    "input",
            expected: "result",
            wantErr:  false,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            mockObj := tt.setupMock()
            // Test implementation
            if mockWithExpectations, ok := mockObj.(*MockExample); ok {
                mockWithExpectations.AssertExpectations(t)
            }
        })
    }
}
```

#### Prohibited Patterns
```go
// ❌ DO NOT create factory functions for mocks
func createMockExample() *MockExample {
    return &MockExample{}
}

// ❌ DO NOT create helper functions for mock initialization
func newMockExampleWithDefaults() *MockExample {
    mock := &MockExample{}
    mock.On("SomeMethod", mock.Anything).Return(nil)
    return mock
}
```

### Mock Usage Guidelines
1. **Implementation Flexibility**: Mock structures can use either `mock.Mock` embedding or function fields based on complexity requirements
2. **Testify/Mock Methods**: When using `mock.Mock`, all mock methods MUST use `m.Called()` for proper argument capture and return value handling
3. **Function Field Methods**: When using function fields, check if the function is not nil before calling it
4. **Direct Initialization**: Initialize mocks directly in test functions using `&MockStruct{}`
5. **No Factory Functions**: Do not create `createMock...()` or `newMock...()` helper functions
6. **Per-Test Configuration**: Configure mock expectations within each test function or test case setup
7. **Expectation Verification**: When using testify/mock, always call `mockObj.AssertExpectations(t)`

Follow these rules to generate consistent test code.