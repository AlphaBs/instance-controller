package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/ksi/instance-controller/docs"
	"github.com/ksi/instance-controller/internal/config"
)

func NewRouter(client EC2Client, cfg config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authorized := router.Group("/")
	authorized.Use(basicAuth(cfg.BasicAuthUser, cfg.BasicAuthPassword))
	authorized.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	handler := NewHandler(client, cfg.EC2InstanceID)
	v1 := authorized.Group("/api/v1")
	v1.GET("/instance", handler.GetInstance)
	v1.POST("/instance/state", handler.ChangeInstanceState)

	return router
}

func basicAuth(expectedUser, expectedPassword string) gin.HandlerFunc {
	expectedUserHash := sha256.Sum256([]byte(expectedUser))
	expectedPasswordHash := sha256.Sum256([]byte(expectedPassword))

	return func(c *gin.Context) {
		user, password, ok := c.Request.BasicAuth()
		userHash := sha256.Sum256([]byte(user))
		passwordHash := sha256.Sum256([]byte(password))
		valid := ok && subtle.ConstantTimeCompare(userHash[:], expectedUserHash[:]) == 1 &&
			subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1
		if !valid {
			c.Header("WWW-Authenticate", `Basic realm="instance-controller", charset="UTF-8"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		c.Next()
	}
}
