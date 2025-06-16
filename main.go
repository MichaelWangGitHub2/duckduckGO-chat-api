package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Configuration du serveur
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Mode de debug depuis les variables d'environnement
	if os.Getenv("GIN_MODE") != "release" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialisation du router
	router := gin.Default()

	// Configuration CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Routes de l'API
	api := router.Group("/api/v1")
	{
		// Routes essentielles du chat IA
		api.GET("/health", HealthCheck)
		api.GET("/models", GetModels)
		api.POST("/chat/completions", ChatHandler)
		api.POST("/chat/stream", StreamChatHandler)
		api.DELETE("/chat/clear", ClearChatHandler)
	}

	// Route racine pour information
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":        "DuckDuckGo Chat API",
			"version":     "1.0.0",
			"description": "API minimaliste pour DuckDuckGo Chat IA",
			"endpoints": gin.H{
				"health":      "GET /api/v1/health",
				"models":      "GET /api/v1/models",
				"chat":        "POST /api/v1/chat/completions",
				"chat_stream": "POST /api/v1/chat/stream",
				"clear":       "DELETE /api/v1/chat/clear",
			},
		})
	})

	log.Printf("🚀 DuckDuckGo Chat API démarrée sur le port %s", port)
	log.Printf("📋 Documentation API disponible sur http://localhost:%s/", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("❌ Erreur de démarrage du serveur:", err)
	}
}
