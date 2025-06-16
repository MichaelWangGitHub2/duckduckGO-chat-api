package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Types et structures de base
type Model string

const (
	GPT4Mini Model = "gpt-4o-mini"
	Claude3  Model = "claude-3-haiku-20240307"
	Llama    Model = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	Mixtral  Model = "mistralai/Mistral-Small-24B-Instruct-2501"
	O4Mini   Model = "o4-mini"
)

const (
	StatusURL = "https://duckduckgo.com/duckchat/v1/status"
	ChatURL   = "https://duckduckgo.com/duckchat/v1/chat"
)

// Structures pour l'API
type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type ToolChoice struct {
	NewsSearch      bool `json:"NewsSearch"`
	VideosSearch    bool `json:"VideosSearch"`
	LocalSearch     bool `json:"LocalSearch"`
	WeatherForecast bool `json:"WeatherForecast"`
}

type Metadata struct {
	ToolChoice ToolChoice `json:"toolChoice"`
}

type ChatPayload struct {
	Model       Model     `json:"model"`
	Metadata    Metadata  `json:"metadata"`
	Messages    []Message `json:"messages"`
	CanUseTools bool      `json:"canUseTools"`
}

type DynamicHeaders struct {
	FeSignals string
	FeVersion string
	VqdHash1  string
}

// Structure principale du chat
type ChatSession struct {
	OldVqd     string
	NewVqd     string
	Model      Model
	Messages   []Message
	Client     *http.Client
	CookieJar  *cookiejar.Jar
	RetryCount int
	FeSignals  string
	FeVersion  string
	VqdHash1   string
}

// Fonction pour obtenir le token VQD
func GetVQD() string {
	client := &http.Client{Timeout: 10 * time.Second}

	// Configuration des cookies
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "duckassist-opt-in-count", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "isRecentChatOn", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "preferredDuckAiModel", Value: "3", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)
	client.Jar = jar

	req, _ := http.NewRequest("GET", StatusURL, nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("DNT", "1")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Sec-CH-UA", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("x-vqd-accept", "1")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Erreur lors de la récupération du VQD: %v", err)
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get("x-vqd-4")
}

// Extraction des headers dynamiques (valeurs fixes fonctionnelles)
func GetDynamicHeaders() *DynamicHeaders {
	return &DynamicHeaders{
		FeSignals: "eyJzdGFydCI6MTc0OTgyODU3NzE1NiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjYwfV0sImVuZCI6NTM4MX0=",
		FeVersion: "serp_20250613_094749_ET-cafd73f97f51c983eb30",
		VqdHash1:  "eyJzZXJ2ZXJfaGFzaGVzIjpbIm5oWlUrcVZ3d3dzODFPVStDTm4vVkZJcS9DbXBSeGxYY2E5cHpGQ0JVZUk9IiwiajRNNmNBRzRheVFqQ21kWkN0a1IzOFY3eVRpd1gvZ2RmcDFueFhEdlV3cz0iXSwiY2xpZW50X2hhc2hlcyI6WyJpRTNqeXRnSm0xZGJaZlo1bW81M1NmaVAxdXUxeEdzY0F5RnB3V2NVOUtrPSIsInJaRGtaR2h4S0JEL1JuY00xVVNraHZNM3pLdEJzQmlzSlJTWFF4L2QzRFU9Il0sInNpZ25hbHMiOnt9LCJtZXRhIjp7InYiOiIzIiwiY2hhbGxlbmdlX2lkIjoiODU3NjA5YjlmMTg2NThlMWM0MzZhZWI2MGM0MDc1ZjdhYWNmYmI0OTlhY2Y4NTVmNDJkNWRjZmM5MTViNDhiOGg4amJ0IiwidGltZXN0YW1wIjoiMTc0OTgyODU3NjQ5NyIsIm9yaWdpbiI6Imh0dHBzOi8vZHVja2R1Y2tnby5jb20iLCJzdGFjayI6IkVycm9yXG5hdCBiYSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LmNhZmQ3M2Y5N2Y1MWM5ODNlYjMwLmpzOjE6NzQ4MDMpXG5hdCBhc3luYyBkaXNwYXRjaFNlcnZpY2VJbml0aWFsVlFEIChodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuY2FmZDczZjk3ZjUxYzk4M2ViMzAuanM6MTo5OTUyOSkifX0=",
	}
}

// Initialisation d'une nouvelle session de chat
func NewChatSession(model Model) *ChatSession {
	vqd := GetVQD()
	if vqd == "" {
		log.Printf("⚠️ Impossible d'obtenir le token VQD")
		return nil
	}

	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "duckassist-opt-in-count", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "isRecentChatOn", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "preferredDuckAiModel", Value: "3", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)

	headers := GetDynamicHeaders()

	return &ChatSession{
		OldVqd:     vqd,
		NewVqd:     vqd,
		Model:      model,
		Messages:   []Message{},
		CookieJar:  jar,
		Client:     &http.Client{Timeout: 30 * time.Second, Jar: jar},
		RetryCount: 0,
		FeSignals:  headers.FeSignals,
		FeVersion:  headers.FeVersion,
		VqdHash1:   headers.VqdHash1,
	}
}

// Envoi d'une requête de chat
func (c *ChatSession) SendMessage(content string) (*http.Response, error) {
	if c.NewVqd == "" {
		c.NewVqd = GetVQD()
		if c.NewVqd == "" {
			return nil, fmt.Errorf("impossible d'obtenir le token VQD")
		}
	}

	// Ajouter le message de l'utilisateur
	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: content,
	})

	payload := ChatPayload{
		Model: c.Model,
		Metadata: Metadata{
			ToolChoice: ToolChoice{
				NewsSearch:      false,
				VideosSearch:    false,
				LocalSearch:     false,
				WeatherForecast: false,
			},
		},
		Messages:    c.Messages,
		CanUseTools: true,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la sérialisation: %v", err)
	}

	req, err := http.NewRequest("POST", ChatURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %v", err)
	}

	// Configuration des headers (rétro-ingénierie)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DNT", "1")
	req.Header.Set("Origin", "https://duckduckgo.com")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Sec-CH-UA", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("x-fe-signals", c.FeSignals)
	req.Header.Set("x-fe-version", c.FeVersion)
	req.Header.Set("x-vqd-4", c.NewVqd)

	if c.VqdHash1 != "" {
		req.Header.Set("x-vqd-hash-1", c.VqdHash1)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'envoi: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Gestion de l'erreur 418 (Anti-bot) avec retry automatique
		if resp.StatusCode == 418 || resp.StatusCode == 429 || strings.Contains(string(body), "ERR_INVALID_VQD") {
			time.Sleep(2 * time.Second)

			// Rafraîchissement du token VQD
			c.NewVqd = GetVQD()

			// Retry si possible
			if c.NewVqd != "" && c.RetryCount < 3 {
				c.RetryCount++
				log.Printf("🔄 Retry automatique (tentative %d/3)...", c.RetryCount)
				return c.SendMessage(content)
			}
		}
		return nil, fmt.Errorf("erreur %d: %s. Body: %s", resp.StatusCode, resp.Status, string(body))
	}

	// Mise à jour du token VQD pour les prochaines requêtes
	if newVqd := resp.Header.Get("x-vqd-4"); newVqd != "" {
		c.OldVqd = c.NewVqd
		c.NewVqd = newVqd
	}

	c.RetryCount = 0
	return resp, nil
}

// Traitement du streaming de réponse
func (c *ChatSession) ProcessStreamResponse(resp *http.Response) (chan string, chan error) {
	stream := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer resp.Body.Close()
		defer close(stream)
		defer close(errChan)

		scanner := bufio.NewScanner(resp.Body)
		var responseBuffer strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			if line == "data: [DONE]" {
				break
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var messageData struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal([]byte(data), &messageData); err != nil {
					log.Printf("Erreur unmarshaling: %v", err)
					continue
				}

				if messageData.Message != "" {
					stream <- messageData.Message
					responseBuffer.WriteString(messageData.Message)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("erreur lecture stream: %v", err)
			return
		}

		// Ajouter la réponse complète à l'historique
		if responseBuffer.Len() > 0 {
			c.Messages = append(c.Messages, Message{
				Role:    "assistant",
				Content: responseBuffer.String(),
			})
		}
	}()

	return stream, errChan
}

// Nettoyage de la session
func (c *ChatSession) Clear() {
	c.Messages = []Message{}
	c.NewVqd = GetVQD()
	c.OldVqd = c.NewVqd
	c.RetryCount = 0
}
