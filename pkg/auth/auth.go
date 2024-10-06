package auth

import (
 "time"
 "fmt"

 "github.com/golang-jwt/jwt"
)

// Secret key for signing JWTs (keep this secret!)
var jwtKey = []byte("your_secret_key") // Replace with a strong, randomly generated key

// Claims struct to define the claims in the JWT
type Claims struct {
 Username string `json:"username"`
 Role     string `json:"role"` // e.g., "supplier" or "user"
 jwt.StandardClaims
}

// GenerateJWT generates a new JWT token for the given user
func GenerateJWT(username, role string) (string, error) {
    expirationTime := time.Now().Add(1 * time.Hour) // Token expires in 1 hour
    claims := &Claims{
    Username: username,
    Role:     role,
    StandardClaims: jwt.StandardClaims{
    ExpiresAt: expirationTime.Unix(),
    },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(jwtKey)
    if err != nil {
    return "", err
    }

    return tokenString, nil
}

// ValidateJWT validates the given JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
 claims := &Claims{}
 token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
  if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
   return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
  }
  return jwtKey, nil
 })

 if err != nil {
  return nil, err
 }

 if claims, ok := token.Claims.(*Claims); ok && token.Valid {
  return claims, nil
 }

 return nil, fmt.Errorf("invalid token")
}