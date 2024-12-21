package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type ReplyClaims struct {
	UserID  uint `json:"user_id"`
	EventID uint `json:"event_id"`
	jwt.RegisteredClaims
}

func CreateReplyToken(eventId uint, userId uint) (string, error) {
	claims := ReplyClaims{
		UserID:  userId,
		EventID: eventId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "messaging",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tks, err := token.SignedString([]byte(viper.GetString("security.reply_token_secret")))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}
	return tks, nil
}

func ParseReplyToken(tk string) (ReplyClaims, error) {
	var claims ReplyClaims
	token, err := jwt.ParseWithClaims(tk, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method)
		}
		return []byte(viper.GetString("security.reply_token_secret")), nil
	})
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, fmt.Errorf("invalid token")
	}
	return claims, nil
}
