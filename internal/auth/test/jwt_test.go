package auth

import (
	"testing"
	"time"

	"datacenter/internal/auth"
)

func TestJWTService(t *testing.T) {
	secretKey := "test-secret-key"
	tokenExpiration := 1 * time.Hour
	refreshExpiration := 24 * time.Hour

	jwtService := auth.NewJWTService(secretKey, tokenExpiration, refreshExpiration)

	userID := "test-user-1"
	roles := []string{"dataowner"}
	permissions := []string{"crud_data"}

	// 测试生成Token
	token, err := jwtService.GenerateToken(userID, roles, permissions)
	if err != nil {
		t.Errorf("GenerateToken() error = %v", err)
		return
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
		return
	}

	// 测试验证Token
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
		return
	}

	if claims.UserID != userID {
		t.Errorf("ValidateToken() userID = %v, want %v", claims.UserID, userID)
	}

	if len(claims.Roles) != len(roles) {
		t.Errorf("ValidateToken() roles length = %d, want %d", len(claims.Roles), len(roles))
	}

	if len(claims.Permissions) != len(permissions) {
		t.Errorf("ValidateToken() permissions length = %d, want %d", len(claims.Permissions), len(permissions))
	}

	// 测试刷新Token
	refreshedToken, err := jwtService.RefreshToken(token)
	if err != nil {
		t.Errorf("RefreshToken() error = %v", err)
		return
	}

	if refreshedToken == "" {
		t.Error("RefreshToken() returned empty token")
		return
	}

	if refreshedToken == token {
		t.Error("RefreshToken() returned the same token")
		return
	}

	// 验证刷新后的Token
	refreshedClaims, err := jwtService.ValidateToken(refreshedToken)
	if err != nil {
		t.Errorf("ValidateToken() error for refreshed token = %v", err)
		return
	}

	if refreshedClaims.UserID != userID {
		t.Errorf("ValidateToken() userID for refreshed token = %v, want %v", refreshedClaims.UserID, userID)
	}
}

func TestHashPassword(t *testing.T) {
	password := "test-password"
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		t.Errorf("HashPassword() error = %v", err)
		return
	}

	if hashedPassword == "" {
		t.Error("HashPassword() returned empty hash")
		return
	}

	// 测试密码验证
	err = auth.CheckPassword(password, hashedPassword)
	if err != nil {
		t.Errorf("CheckPassword() error = %v", err)
		return
	}

	// 测试错误密码
	err = auth.CheckPassword("wrong-password", hashedPassword)
	if err == nil {
		t.Error("CheckPassword() should return error for wrong password")
		return
	}
}
