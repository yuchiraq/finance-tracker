// auth/auth.go
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	username  = "boss"
	password  = "0162"
	authToken = "my-secret-token-123"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем куки
		token, err := c.Cookie("auth_token")
		if err == nil && token == authToken {
			c.Next()
			return
		}

		// Проверяем логин и пароль
		user, pass, hasAuth := c.Request.BasicAuth()
		if hasAuth && user == username && pass == password {
			// Успешная авторизация, устанавливаем куки
			c.SetCookie("auth_token", authToken, 3600*24*30, "/", "", false, true) // Куки на 30 дней
			c.Next()
			return
		}

		// Запрашиваем авторизацию
		c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
