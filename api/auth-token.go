package api

import (
	"net/http"
	"ocs-ad-inventorymanagement/client"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(getJWTSecret())

func getJWTSecret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "supersecretjwtkey"
	}
	return s
}

type AuthTokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthTokenResponse struct {
	Token string `json:"token"`
}

// POST /auth-token
func AuthTokenHandler(c *gin.Context) {
	var req AuthTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username dan password wajib diisi"})
		return
	}
	ocsCfg := client.LoadOCSAuthConfig()
	if err := client.AuthenticateOCSWeb(ocsCfg.OCSURL, req.Username, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Success, generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(3 * time.Minute).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal generate token"})
		return
	}
	c.JSON(http.StatusOK, AuthTokenResponse{Token: tokenString})
}
