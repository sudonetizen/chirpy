package auth

import (
    "fmt"
    "time"
    "github.com/google/uuid"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

func HashPassword(p string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)

    if err != nil {return "", err }

    return string(hash), nil
}

func CheckPasswordHash(p, h string) error {
    err := bcrypt.CompareHashAndPassword([]byte(h), []byte(p))
    
    if err != nil {return err}

    return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
    claims := &jwt.RegisteredClaims{
        Issuer: "chirpy",
        IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
        ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
        Subject: userID.String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    ss, err := token.SignedString([]byte(tokenSecret))

    if err != nil {return "", err}
    
    return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
    token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {return []byte(tokenSecret), nil})
    if err != nil {return uuid.Nil, err}
    
    claims, ok := token.Claims.(*jwt.RegisteredClaims)
    if !ok {return uuid.Nil, fmt.Errorf("invalid token\n")}

    userID, err := uuid.Parse(claims.Subject)
    if err != nil {return uuid.Nil, err}

    return userID, nil
}
