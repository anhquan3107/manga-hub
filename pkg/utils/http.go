package utils

import "github.com/gin-gonic/gin"

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func OK(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
