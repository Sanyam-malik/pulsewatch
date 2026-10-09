package auth

import (
	"github.com/sanyam-malik/pulsewatch/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MiddlewareProvider holds all middleware functions
type MiddlewareProvider struct {
	tokenMaker *TokenMaker
	repo       Repository
}

// NewMiddlewareProvider creates a new middleware provider
func NewMiddlewareProvider(tokenMaker *TokenMaker, repo Repository) *MiddlewareProvider {
	return &MiddlewareProvider{
		tokenMaker: tokenMaker,
		repo:       repo,
	}
}

// Auth is a middleware that verifies JWT access tokens only
func (p *MiddlewareProvider) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.NewFailResponse("Authorization header is required"))
			c.Abort()
			return
		}

		// JWT authentication
		// Add Bearer prefix if not present
		if !strings.HasPrefix(authHeader, "Bearer ") {
			authHeader = "Bearer " + authHeader
		}

		// Check if the header has the Bearer prefix
		fields := strings.Fields(authHeader)
		if len(fields) != 2 || fields[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, utils.NewFailResponse("Invalid authorization header format"))
			c.Abort()
			return
		}

		// Extract the token
		accessToken := fields[1]

		// Verify the token
		claims, err := p.tokenMaker.VerifyToken(c.Request.Context(), accessToken, "access")
		if err != nil {
			c.JSON(http.StatusUnauthorized, utils.NewFailResponse("Invalid or expired token"))
			c.Abort()
			return
		}

		// Check if it's an access token
		if claims.Type != "access" {
			c.JSON(http.StatusUnauthorized, utils.NewFailResponse("Invalid token type"))
			c.Abort()
			return
		}

		user, err := p.repo.FindByID(c.Request.Context(), claims.UserID)
		if err != nil || user == nil || !user.Active {
			c.JSON(http.StatusUnauthorized, utils.NewFailResponse("Account is unavailable"))
			c.Abort()
			return
		}
		groupID := c.GetHeader("X-Group-ID")
		if groupID == "" {
			groupID = claims.GroupID
		}
		membership, err := p.repo.ResolveMembership(c.Request.Context(), user.ID, groupID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.NewFailResponse("Unable to verify group membership"))
			c.Abort()
			return
		}
		if membership == nil && groupID == "" {
			membership, err = p.repo.ResolveMembership(c.Request.Context(), user.ID, "")
			if err != nil {
				c.JSON(http.StatusInternalServerError, utils.NewFailResponse("Unable to verify group membership"))
				c.Abort()
				return
			}
		}
		if membership == nil {
			c.JSON(http.StatusForbidden, utils.NewFailResponse("You are not a member of this group"))
			c.Abort()
			return
		}

		// Viewers may inspect the application but cannot modify monitored data.
		if membership.Role == RoleViewer && isWriteRequest(c) {
			c.JSON(http.StatusForbidden, utils.NewFailResponse("Viewer access is read-only"))
			c.Abort()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/settings") &&
			isWriteRequest(c) && membership.Role != RoleOwner && membership.Role != RoleAdmin {
			c.JSON(http.StatusForbidden, utils.NewFailResponse("Group administrator access required for system settings"))
			c.Abort()
			return
		}

		// Set current identity in both Gin and standard request contexts. Domain
		// repositories can consume auth.GroupIDFromContext without importing Gin.
		c.Set("userId", claims.UserID)
		c.Set("email", user.Email)
		c.Set("authType", "jwt")
		c.Set("groupId", membership.GroupID)
		c.Set("role", membership.Role)
		c.Request = c.Request.WithContext(WithIdentity(c.Request.Context(), membership.GroupID, membership.Role))

		c.Next()
	}
}

func isWriteRequest(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	// Password and 2FA controls are personal account security, not workspace
	// mutations, so read-only members retain access to those routes.
	return !strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/")
}
