package services

import (
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"forma3d/server/internal/models"
)

// tokenTTL — время жизни JWT администратора.
const tokenTTL = 24 * time.Hour

// maxPasswordLen — верхняя граница длины пароля на входе.
const maxPasswordLen = 200

// AuthService — логин единственного администратора и проверка JWT (HS256).
type AuthService struct {
	adminEmail        string
	adminPasswordHash []byte
	jwtSecret         []byte
}

// NewAuthService создаёт сервис авторизации.
func NewAuthService(adminEmail, adminPasswordHash string, jwtSecret []byte) *AuthService {
	return &AuthService{
		adminEmail:        strings.ToLower(adminEmail),
		adminPasswordHash: []byte(adminPasswordHash),
		jwtSecret:         jwtSecret,
	}
}

// Login сверяет email и bcrypt-хеш пароля, выдаёт JWT с exp 24h.
func (s *AuthService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	if err := validateEmail(req.Email); err != nil {
		return nil, ErrInvalidCredentials
	}
	if req.Password == "" || len(req.Password) > maxPasswordLen {
		return nil, ErrInvalidCredentials
	}
	// Обе проверки выполняются всегда — не даём тайминг-оракула по email.
	emailOK := subtle.ConstantTimeCompare([]byte(strings.ToLower(req.Email)), []byte(s.adminEmail)) == 1
	passErr := bcrypt.CompareHashAndPassword(s.adminPasswordHash, []byte(req.Password))
	if !emailOK || passErr != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expiresAt := now.Add(tokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  s.adminEmail,
		"role": "admin",
		"iat":  now.Unix(),
		"exp":  expiresAt.Unix(),
	})
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	return &models.LoginResponse{Token: signed, ExpiresAt: expiresAt}, nil
}

// ValidateToken проверяет подпись (только HS256) и срок действия JWT.
func (s *AuthService) ValidateToken(tokenString string) error {
	_, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return fmt.Errorf("validate token: %w", err)
	}
	return nil
}
