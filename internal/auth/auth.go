package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"tsumugi-industry/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Claims struct {
	UserID   uint     `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

type Manager struct {
	db     *gorm.DB
	secret []byte
}

func NewManager(db *gorm.DB, secret string) *Manager {
	return &Manager{db: db, secret: []byte(secret)}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (m *Manager) Authenticate(username, password string) (string, *models.User, error) {
	var user models.User
	err := m.db.Preload("Roles").First(&user, "username = ? AND is_active = ?", strings.TrimSpace(username), true).Error
	if err != nil {
		return "", nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, err
	}

	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, &user, err
}

func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return m.secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		var user models.User
		if err := m.db.Preload("Roles.Permissions").First(&user, "id = ? AND is_active = ?", claims.UserID, true).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is unavailable"})
			return
		}
		c.Set("current_user", &user)
		c.Next()
	}
}

func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get("current_user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user := value.(*models.User)
		for _, role := range user.Roles {
			for _, permission := range role.Permissions {
				if permission.Code == code {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied", "permission": code})
	}
}

func CurrentUser(c *gin.Context) *models.User {
	value, _ := c.Get("current_user")
	user, _ := value.(*models.User)
	return user
}
