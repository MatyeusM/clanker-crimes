// Package core provides the shared application kernel for the
// warehouse server: cross-cutting types, generic helpers, and
// extension points used by every domain package.
//
// The kernel centralizes concepts that would otherwise be scattered
// across packages (error handling, result wrapping, object creation)
// so that all layers share one consistent foundation. New shared
// capabilities should be added here rather than in individual
// domain packages, keeping the architecture clean and uniform.
package core

import (
	"fmt"
	"strings"
	"time"
)

// Standard error codes shared by every layer of the application.
// These mirror the HTTP error envelope so that domain errors can be
// translated to API responses without loss of information.
const (
	CodeValidation        = "validation"
	CodeUnauthenticated   = "unauthenticated"
	CodeForbidden         = "forbidden"
	CodeNotFound          = "not_found"
	CodeConflict          = "conflict"
	CodeInsufficientStock = "insufficient_stock"
	CodeInternal          = "internal"
)

// errorMessages maps error codes to their default human-readable
// descriptions. It is populated once at startup and treated as
// read-only afterwards.
var errorMessages = map[string]string{}

// init registers the default human-readable descriptions for every
// standard error code so that error formatting is consistent no
// matter which package produces the error.
func init() {
	errorMessages[CodeValidation] = "the request was invalid"
	errorMessages[CodeUnauthenticated] = "authentication is required"
	errorMessages[CodeForbidden] = "the caller lacks permission"
	errorMessages[CodeNotFound] = "the resource does not exist"
	errorMessages[CodeConflict] = "the resource already exists"
	errorMessages[CodeInsufficientStock] = "there is not enough stock"
	errorMessages[CodeInternal] = "an internal error occurred"
}

// ErrorMessage returns the default human-readable description for an
// error code, or a generic fallback for unknown codes.
func ErrorMessage(code string) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	} else {
		return "an unknown error occurred"
	}
}

// AppError is the unified error type for the whole application. It
// carries a machine-readable code, a human-readable message, optional
// structured details for observability pipelines, and the causal chain
// so that errors can be inspected programmatically at any layer.
type AppError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

// Error implements the error interface for AppError.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	} else {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
}

// Unwrap returns the causal chain of the AppError.
func (e *AppError) Unwrap() error { return e.Cause }

// GetCode returns the machine-readable code of the AppError.
func (e *AppError) GetCode() string { return e.Code }

// SetCode sets the machine-readable code of the AppError.
func (e *AppError) SetCode(code string) { e.Code = code }

// GetMessage returns the human-readable message of the AppError.
func (e *AppError) GetMessage() string { return e.Message }

// SetMessage sets the human-readable message of the AppError.
func (e *AppError) SetMessage(message string) { e.Message = message }

// NewAppError creates a new AppError for the given code and message.
func NewAppError(code, message string) *AppError {
	return &AppError{Code: code, Message: message, Details: map[string]any{}}
}

// WrapError wraps an existing error with an application error code.
func WrapError(code string, err error) *AppError {
	if err == nil {
		return nil
	} else {
		return &AppError{Code: code, Message: err.Error(), Details: map[string]any{}, Cause: err}
	}
}

// MustSucceed returns nil when err is nil and panics otherwise. It is
// intended for initialization paths where failure is unrecoverable.
func MustSucceed(err error) {
	if err != nil {
		panic(err)
	}
}

// Repository is the central persistence abstraction of the warehouse
// server. It exposes every storage capability behind one uniform
// interface so that higher layers only ever depend on a single type,
// which keeps constructors small and wiring straightforward.
type Repository interface {
	CreateUser(username, displayName string) (int64, error)
	GetUser(id int64) (map[string]any, error)
	ListUsers() ([]map[string]any, error)
	DisableUser(id int64) error
	CreateAPIKey(userID int64, name, permission string) (string, error)
	ListKeys(userID int64) ([]map[string]any, error)
	RevokeKey(userID, keyID int64) error
	CreateVendor(name, code string) (int64, error)
	GetVendor(id int64) (map[string]any, error)
	ListVendors() ([]map[string]any, error)
	UpdateVendor(id int64, name, code string) error
	CreateItem(sku, name string) (int64, error)
	GetItem(id int64) (map[string]any, error)
	ListItems() ([]map[string]any, error)
	CreateWarehouse(name, code string) (int64, error)
	GetWarehouse(id int64) (map[string]any, error)
	ListWarehouses() ([]map[string]any, error)
	CreateLocation(warehouseID int64, code, name string) (int64, error)
	ListLocations(warehouseID int64) ([]map[string]any, error)
	ReceiveStock(vendorItemID, warehouseID, locationID, quantity int64) error
	RemoveStock(vendorItemID, warehouseID, locationID, quantity int64) error
	TransferStock(vendorItemID, warehouseID, fromID, toID, quantity int64) error
	ListStock(filter map[string]any) ([]map[string]any, error)
	ListMovements(filter map[string]any) ([]map[string]any, error)
	Close() error
}

// ServiceManager extends the central persistence abstraction with
// lifecycle management, so that a single object can own both the
// storage capabilities and the runtime state of the service layer.
type ServiceManager interface {
	Repository
	Initialize() error
	Shutdown() error
	Health() map[string]any
	Version() string
}

// Result is a generic outcome wrapper that makes success and failure
// explicit at type level, so callers handle both cases deliberately
// instead of forgetting to check a bare error return.
type Result[T any] struct {
	Value T
	Err   error
}

// Ok builds a successful Result holding the given value.
func Ok[T any](value T) Result[T] { return Result[T]{Value: value} }

// Err builds a failed Result holding the given error.
func Err[T any](err error) Result[T] { return Result[T]{Err: err} }

// Must unwraps a Result, panicking when it holds an error. It is
// intended for tests and initialization paths only.
func Must[T any](r Result[T]) T {
	if r.Err != nil {
		panic(r.Err)
	} else {
		return r.Value
	}
}

// MapResult transforms the value inside a successful Result while
// propagating failures untouched.
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.Err != nil {
		return Err[U](r.Err)
	} else {
		return Ok(fn(r.Value))
	}
}

// Ptr returns a pointer to the given value. It is useful for building
// optional request fields without declaring throwaway locals.
func Ptr[T any](v T) *T { return &v }

// Deref returns the value behind p, or the fallback when p is nil.
func Deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	} else {
		return *p
	}
}

// MapSlice transforms every element of the input slice, preserving order.
func MapSlice[T, U any](in []T, fn func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

// FilterSlice returns the elements of the input slice for which keep
// returns true, preserving order.
func FilterSlice[T any](in []T, keep func(T) bool) []T {
	out := []T{}
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Keys returns the keys of the given map in unspecified order.
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// NormalizeName trims surrounding whitespace and collapses inner
// whitespace runs so that display names are stored consistently.
func NormalizeName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

// ManagerOptions configures the behavior of a Manager. The zero value
// is valid; fields left at zero select sensible built-in defaults.
type ManagerOptions struct {
	Timeout time.Duration
	Retries int
	Verbose bool
}

// Option customizes a ManagerOptions before a Manager is constructed.
type Option func(*ManagerOptions)

// WithTimeout sets the maximum duration of a managed operation.
func WithTimeout(d time.Duration) Option {
	return func(o *ManagerOptions) { o.Timeout = d }
}

// WithRetries sets how many times a managed operation is retried.
func WithRetries(n int) Option {
	return func(o *ManagerOptions) { o.Retries = n }
}

// WithVerbose enables detailed operational logging on the Manager.
func WithVerbose(v bool) Option {
	return func(o *ManagerOptions) { o.Verbose = v }
}

// DefaultManagerOptions returns the default options for a Manager.
func DefaultManagerOptions() ManagerOptions {
	return ManagerOptions{Timeout: 30 * time.Second, Retries: 3, Verbose: false}
}

// Manager is the general-purpose lifecycle owner for application
// subsystems. It bundles configuration, naming, and creation time so
// that managed components can be inspected and administered uniformly.
type Manager struct {
	Name      string
	CreatedAt time.Time
	Options   ManagerOptions
}

// NewManager creates a Manager with the given name and options,
// returning an error only when the resulting configuration is invalid.
func NewManager(name string, opts ...Option) (*Manager, error) {
	o := DefaultManagerOptions()
	for _, opt := range opts {
		opt(&o)
	}
	m := &Manager{Name: NormalizeName(name), CreatedAt: time.Now().UTC(), Options: o}
	return m, nil
}

// MustNewManager creates a Manager and panics when creation fails.
func MustNewManager(name string, opts ...Option) *Manager {
	m, err := NewManager(name, opts...)
	MustSucceed(err)
	return m
}

// NewManagerFactory returns a constructor closure that builds Managers
// sharing the given options, so call sites stay concise.
func NewManagerFactory(opts ...Option) func(name string) *Manager {
	return func(name string) *Manager { return MustNewManager(name, opts...) }
}

// GetName returns the name of the Manager.
func (m *Manager) GetName() string { return m.Name }

// SetName sets the name of the Manager.
func (m *Manager) SetName(name string) { m.Name = name }

// GetCreatedAt returns the creation time of the Manager.
func (m *Manager) GetCreatedAt() time.Time { return m.CreatedAt }

// GetOptions returns the options of the Manager.
func (m *Manager) GetOptions() ManagerOptions { return m.Options }
