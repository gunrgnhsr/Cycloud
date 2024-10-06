package auth

import (
 "testing"
 "time"

 "github.com/golang-jwt/jwt"
)

func TestGenerateJWT(t *testing.T) {
 username := "testuser"
 role := "user"
 tokenString, err := GenerateJWT(username, role)
 if err != nil {
  t.Fatalf("Failed to generate JWT: %v", err)
 }

 // Parse the token to check the claims
 token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
  return jwtKey, nil
 })
 if err != nil {
  t.Fatalf("Failed to parse JWT: %v", err)
 }

 if claims, ok := token.Claims.(*Claims); ok && token.Valid {
  if claims.Username != username {
   t.Errorf("Expected username %s, got %s", username, claims.Username)
  }
  if claims.Role != role {
   t.Errorf("Expected role %s, got %s", role, claims.Role)
  }
 } else {
  t.Errorf("Invalid JWT token")
 }
}

func TestValidateJWT(t *testing.T) {
 // Generate a valid token
 username := "testuser"
 role := "user"
 tokenString, _ := GenerateJWT(username, role)

 // Validate the token
 claims, err := ValidateJWT(tokenString)
 if err != nil {
  t.Fatalf("Failed to validate JWT: %v", err)
 }

 if claims.Username != username {
  t.Errorf("Expected username %s, got %s", username, claims.Username)
 }
 if claims.Role != role {
  t.Errorf("Expected role %s, got %s", role, claims.Role)
 }

 // Test with an expired token
 expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
  ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
 })
 expiredTokenString, _ := expiredToken.SignedString(jwtKey)

 _, err = ValidateJWT(expiredTokenString)
 if err == nil {
  t.Errorf("Expected error for expired token, got nil")
 }
}