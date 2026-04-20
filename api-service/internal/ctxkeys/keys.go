// Package ctxkeys defines context key constants used across api-service
// to prevent circular imports between the auth and user packages.
package ctxkeys

// UserID is the Gin context key under which the authenticated user ID is stored.
const UserID = "userID"
