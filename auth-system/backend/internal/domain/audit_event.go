// internal/domain/audit_event.go
package domain

// EventType is a string constant identifying the type of audit log event.
type EventType string

const (
	// Authentication events
	EventLoginSuccess          EventType = "LOGIN_SUCCESS"
	EventLoginFailure          EventType = "LOGIN_FAILURE"
	EventLogout                EventType = "LOGOUT"
	EventLogoutAll             EventType = "LOGOUT_ALL"
	EventTokenRefreshed        EventType = "TOKEN_REFRESHED"
	EventSuspiciousTokenReuse  EventType = "SUSPICIOUS_TOKEN_REUSE"

	// Account lifecycle events
	EventEmailVerified         EventType = "EMAIL_VERIFIED"
	EventPasswordChanged       EventType = "PASSWORD_CHANGED"
	EventPasswordResetReq      EventType = "PASSWORD_RESET_REQUESTED"
	EventPasswordResetDone     EventType = "PASSWORD_RESET_COMPLETED"
	EventAccountLocked         EventType = "ACCOUNT_LOCKED"
	EventUserInvited           EventType = "USER_INVITED"
	EventUserDisabled          EventType = "USER_DISABLED"
	EventUserDeleted           EventType = "USER_DELETED"

	// RBAC events
	EventRoleAssigned          EventType = "ROLE_ASSIGNED"
	EventRoleUnassigned        EventType = "ROLE_UNASSIGNED"

	// OAuth events
	EventOAuthClientCreated    EventType = "OAUTH_CLIENT_CREATED"
	EventOAuthCodeIssued       EventType = "OAUTH_CODE_ISSUED"
	EventOAuthTokenIssued      EventType = "OAUTH_TOKEN_ISSUED"

	// Social login events
	EventGoogleLinked          EventType = "GOOGLE_ACCOUNT_LINKED"
	EventGoogleLogin           EventType = "GOOGLE_LOGIN"
)
