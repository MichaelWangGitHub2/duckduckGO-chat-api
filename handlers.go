package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Variables globales pour la gestion des sessions
var (
	chatSessions = make(map[string]*ChatSession)
	sessionMutex sync.RWMutex
)

// Structures pour les requêtes/réponses API
type ChatRequest struct {
	Message   string `json:"message" binding:"required"`
	Model     string `json:"model,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type ChatResponse struct {
	Message   string `json:"message"`
	Model     string `json:"model"`
	SessionID string `json:"session_id"`
	Success   bool   `json:"success"`
}

type StreamResponse struct {
	Chunk     string `json:"chunk,omitempty"`
	Done      bool   `json:"done"`
	SessionID string `json:"session_id"`
	Error     string `json:"error,omitempty"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Alias       string `json:"alias"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Success bool   `json:"success"`
}

// Fonction pour obtenir ou créer une session
func getOrCreateSession(sessionID string, model Model) *ChatSession {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if sessionID == "" {
		sessionID = generateSessionID()
	}

	if session, exists := chatSessions[sessionID]; exists {
		if model != "" && session.Model != model {
			session.Model = model
		}
		return session
	}

	if model == "" {
		model = GPT4Mini // Modèle par défaut
	}

	session := NewChatSession(model)
	if session != nil {
		chatSessions[sessionID] = session
	}
	return session
}

// Génération d'un ID de session simple
func generateSessionID() string {
	return fmt.Sprintf("session_%d", len(chatSessions)+1)
}

// Validation du modèle
func validateModel(modelStr string) (Model, error) {
	switch strings.ToLower(modelStr) {
	case "gpt-4o-mini", "gpt4mini", "":
		return GPT4Mini, nil
	case "claude-3-haiku-20240307", "claude", "claude3":
		return Claude3, nil
	case "meta-llama/llama-3.3-70b-instruct-turbo", "llama", "llama3":
		return Llama, nil
	case "mistralai/mistral-small-24b-instruct-2501", "mixtral", "mistral":
		return Mixtral, nil
	case "o4-mini", "o4mini":
		return O4Mini, nil
	default:
		return "", fmt.Errorf("modèle non supporté: %s", modelStr)
	}
}

// Handler pour vérifier la santé de l'API
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "DuckDuckGo Chat API",
		"version":   "1.0.0",
		"timestamp": fmt.Sprintf("%d", getCurrentTimestamp()),
	})
}

func getCurrentTimestamp() int64 {
	return 1749828577156 // Timestamp fixe pour la cohérence
}

// Handler pour obtenir la liste des modèles disponibles
func GetModels(c *gin.Context) {
	models := []ModelInfo{
		{
			ID:          string(GPT4Mini),
			Name:        "GPT-4o Mini",
			Description: "Modèle général rapide et équilibré",
			Alias:       "gpt-4o-mini",
		},
		{
			ID:          string(Claude3),
			Name:        "Claude 3 Haiku",
			Description: "Excellente pour l'écriture créative et les explications",
			Alias:       "claude-3-haiku",
		},
		{
			ID:          string(Llama),
			Name:        "Llama 3.3 70B",
			Description: "Spécialisé en programmation et tâches techniques",
			Alias:       "llama",
		},
		{
			ID:          string(Mixtral),
			Name:        "Mistral Small",
			Description: "Excellent pour l'analyse et le raisonnement",
			Alias:       "mixtral",
		},
		{
			ID:          string(O4Mini),
			Name:        "o4-mini",
			Description: "Très rapide pour les réponses courtes",
			Alias:       "o4mini",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"models":  models,
		"success": true,
		"count":   len(models),
	})
}

// Handler principal pour le chat (réponse complète)
func ChatHandler(c *gin.Context) {
	var req ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   fmt.Sprintf("Requête invalide: %v", err),
			Code:    400,
			Success: false,
		})
		return
	}

	// Validation du modèle
	model, err := validateModel(req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   err.Error(),
			Code:    400,
			Success: false,
		})
		return
	}

	// Obtenir ou créer la session
	session := getOrCreateSession(req.SessionID, model)
	if session == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Impossible de créer la session de chat",
			Code:    500,
			Success: false,
		})
		return
	}

	// Envoyer le message
	resp, err := session.SendMessage(req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   fmt.Sprintf("Erreur de chat: %v", err),
			Code:    500,
			Success: false,
		})
		return
	}
	defer resp.Body.Close()

	// Lire la réponse complète
	var completeResponse strings.Builder
	stream, errChan := session.ProcessStreamResponse(resp)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto done
			}
			completeResponse.WriteString(chunk)
		case err := <-errChan:
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   fmt.Sprintf("Erreur de stream: %v", err),
					Code:    500,
					Success: false,
				})
				return
			}
		}
	}

done:
	// Obtenir l'ID de session
	sessionID := req.SessionID
	if sessionID == "" {
		// Trouver l'ID de session de cette instance
		sessionMutex.RLock()
		for id, sess := range chatSessions {
			if sess == session {
				sessionID = id
				break
			}
		}
		sessionMutex.RUnlock()
	}

	c.JSON(http.StatusOK, ChatResponse{
		Message:   completeResponse.String(),
		Model:     string(session.Model),
		SessionID: sessionID,
		Success:   true,
	})
}

// Handler pour le chat en streaming
func StreamChatHandler(c *gin.Context) {
	var req ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   fmt.Sprintf("Requête invalide: %v", err),
			Code:    400,
			Success: false,
		})
		return
	}

	// Validation du modèle
	model, err := validateModel(req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   err.Error(),
			Code:    400,
			Success: false,
		})
		return
	}

	// Obtenir ou créer la session
	session := getOrCreateSession(req.SessionID, model)
	if session == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Impossible de créer la session de chat",
			Code:    500,
			Success: false,
		})
		return
	}

	// Configuration pour le streaming
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Envoyer le message
	resp, err := session.SendMessage(req.Message)
	if err != nil {
		streamResp := StreamResponse{
			Done:      true,
			SessionID: req.SessionID,
			Error:     fmt.Sprintf("Erreur de chat: %v", err),
		}
		data, _ := json.Marshal(streamResp)
		c.SSEvent("error", string(data))
		return
	}
	defer resp.Body.Close()

	// Obtenir l'ID de session pour la réponse
	sessionID := req.SessionID
	if sessionID == "" {
		sessionMutex.RLock()
		for id, sess := range chatSessions {
			if sess == session {
				sessionID = id
				break
			}
		}
		sessionMutex.RUnlock()
	}

	// Traiter le stream
	stream, errChan := session.ProcessStreamResponse(resp)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Stream terminé
				streamResp := StreamResponse{
					Done:      true,
					SessionID: sessionID,
				}
				data, _ := json.Marshal(streamResp)
				c.SSEvent("done", string(data))
				return
			}

			// Envoyer le chunk
			streamResp := StreamResponse{
				Chunk:     chunk,
				Done:      false,
				SessionID: sessionID,
			}
			data, _ := json.Marshal(streamResp)
			c.SSEvent("chunk", string(data))
			c.Writer.Flush()

		case err := <-errChan:
			if err != nil {
				streamResp := StreamResponse{
					Done:      true,
					SessionID: sessionID,
					Error:     fmt.Sprintf("Erreur de stream: %v", err),
				}
				data, _ := json.Marshal(streamResp)
				c.SSEvent("error", string(data))
				return
			}
		}
	}
}

// Handler pour nettoyer une session de chat
func ClearChatHandler(c *gin.Context) {
	sessionID := c.Query("session_id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "session_id requis",
			Code:    400,
			Success: false,
		})
		return
	}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if session, exists := chatSessions[sessionID]; exists {
		session.Clear()
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Session nettoyée avec succès",
			"session_id": sessionID,
		})
	} else {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Session non trouvée",
			Code:    404,
			Success: false,
		})
	}
}
