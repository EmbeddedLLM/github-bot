package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/wenjielee1/github-bot/models"
)

const BASE_URL = "https://api.jamaibase.com/api/v1/gen_tables"
const MODEL_NAME = "ellm/Qwen/Qwen2.5-72B-w8a8"

// Shared configuration struct for generating responses
var GEN_CONFIG = models.GenConfig{
	EmbeddingModel: "ellm/BAAI/bge-m3",
	Model:          MODEL_NAME,
	Temperature:    0.01,
	MaxTokens:      2000,
	TopP:           0.001,
	RagParams: &models.RagParams{
		K:              5,
		RerankingModel: "ellm/BAAI/bge-reranker-v2-m3",
	},
}

// AuthTransport adds authentication headers to HTTP requests.
type AuthTransport struct {
	Transport http.RoundTripper
	AuthInfo  *models.JamaiAuth
}

// RoundTrip adds the authentication headers to the request.
func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqClone := req.Clone(req.Context())
	reqClone.Header.Set("Authorization", t.AuthInfo.Authorization)
	reqClone.Header.Set("X-PROJECT-ID", t.AuthInfo.XProjectID)
	return t.Transport.RoundTrip(reqClone)
}

// NewJamaiClient creates an HTTP client with authentication headers.
func NewJamaiClient(authInfo *models.JamaiAuth) *http.Client {
	return &http.Client{
		Transport: &AuthTransport{
			Transport: http.DefaultTransport,
			AuthInfo:  authInfo,
		},
	}
}

// sendRequest sends an HTTP request with the specified method, URL, and data.
func sendRequest(client *http.Client, method, url string, data interface{}) (*http.Response, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// CreateKnowledgeTable creates a knowledge table in JAM.AI.
func CreateKnowledgeTable(client *http.Client, tableId string) {
	url := fmt.Sprintf("%s/knowledge", BASE_URL)
	data := models.CreateAgentKnowledgeTableRequest{
		ID:             tableId,
		Cols:           []models.Col{},
		EmbeddingModel: GEN_CONFIG.EmbeddingModel,
	}

	resp, err := sendRequest(client, "POST", url, data)
	if err != nil {
		log.Printf("Error creating knowledge table: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		log.Println("Knowledge table already exists.")
	} else {
		log.Println("Knowledge table created successfully.")
	}
}

// CreateTable creates a table in JAM.AI of the specified type.
func CreateTable(client *http.Client, tableType models.TableType, tableId string, agents []models.Agent) {
	if tableType == models.KnowledgeTable {
		CreateKnowledgeTable(client, tableId)
		return
	}

	url := fmt.Sprintf("%s/%s", BASE_URL, tableType)

	cols := []models.Col{}
	for _, agent := range agents {
		col := models.Col{
			ID:    agent.ColumnID,
			Dtype: "str",
		}
		if len(agent.Messages) > 0 {
			col.GenConfig = &models.GenConfig{
				Model:       GEN_CONFIG.Model,
				Messages:    agent.Messages,
				Temperature: GEN_CONFIG.Temperature,
				MaxTokens:   GEN_CONFIG.MaxTokens,
				TopP:        GEN_CONFIG.TopP,
				RagParams:   nil,
			}
		}
		cols = append(cols, col)
	}

	data := models.CreateAgentChatTableRequest{
		ID:   tableId,
		Cols: cols,
	}

	resp, err := sendRequest(client, "POST", url, data)
	if err != nil {
		log.Fatalf("Error creating chat table: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		log.Println(tableType + " already exists.")
	} else {
		log.Println(tableType + " created successfully.")
	}
}

// NOTE: THESE COMMENTED FUNCTIONS ARE NOT TESTED, ITS JUST A ROUGH IMPLEMENTATION!

// func configureTable(client *http.Client, agentName, characterStory string) {
//     url := fmt.Sprintf("%s/chat/gen_config/update", BASE_URL)
//     data := models.ConfigureAgentChatTableRequest{
//         TableID: fmt.Sprintf("Chat_Ai-Town-%s", agentName),
//         ColumnMap: map[string]models.GenConfig{
//             "AI": {
//                 Model:      GEN_CONFIG.Model,
//                 Messages:   []models.Message{{Role: "system", Content: characterStory}},
//                 Temperature: GEN_CONFIG.Temperature,
//                 MaxTokens:   GEN_CONFIG.MaxTokens,
//                 TopP:        GEN_CONFIG.TopP,
//                 RagParams: models.RagParams{
//                     K:              GEN_CONFIG.RagParams.K,
//                     TableID:        fmt.Sprintf("knowledge_Ai-Town-%s", agentName),
//                     RerankingModel: GEN_CONFIG.RagParams.RerankingModel,
//                 },
//             },
//         },
//     }

//     resp, err := sendRequest(client, "POST", url, data)
//     if err != nil {
//         log.Printf("Error configuring chat table for RAG: %v", err)
//         return
//     }
//     defer resp.Body.Close()

//     log.Println("Chat table configured for RAG successfully.")
// }

// func createAgentConversationTable(client *http.Client, agentChatTable, conversationId string) {
//     url := fmt.Sprintf("%s/chat/duplicate/%s/%s?deploy=true", BASE_URL, agentChatTable, conversationId)

//     req, err := http.NewRequest("POST", url, nil)
//     if err != nil {
//         log.Printf("Error creating request: %v", err)
//         return
//     }

//     req.Header.Set("Accept", "application/json")
//     req.Header.Set("Content-Type", "application/json")

//     resp, err := client.Do(req)
//     if err != nil {
//         log.Printf("Error creating conversation table: %v", err)
//         return
//     }
//     defer resp.Body.Close()

//     if resp.StatusCode == http.StatusConflict {
//         log.Println("Conversation table already exists.")
//     } else {
//         log.Println("Conversation table created successfully.")
//     }
// }

// readAndCollectContent reads the streamed response and collects content when output_column_name matches the specified column.
func readAndCollectContent(resp *http.Response, outputColumn string) (string, error) {
	defer resp.Body.Close()

	var collectedContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
			if line == "[DONE]" {
				break
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				return "", fmt.Errorf("error unmarshaling line: %w", err)
			}

			if outputColumnName, ok := chunk["output_column_name"].(string); ok && outputColumnName == outputColumn {
				if choices, ok := chunk["choices"].([]interface{}); ok {
					for _, choice := range choices {
						if choiceMap, ok := choice.(map[string]interface{}); ok {
							if message, ok := choiceMap["message"].(map[string]interface{}); ok {
								if content, ok := message["content"].(string); ok {
									collectedContent.WriteString(content)
								}
							}
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading response data: %w", err)
	}

	return collectedContent.String(), nil
}

// parseCreateIssueResponse parses the response content into a CreateIssueResponse.
func parseCreateIssueResponse(content string) (models.CreateIssueResponse, error) {
	var issueResponse models.CreateIssueResponse
	if err := json.Unmarshal([]byte(content), &issueResponse); err != nil {
		return issueResponse, fmt.Errorf("error unmarshaling CreateIssueResponse: %w", err)
	}
	return issueResponse, nil
}

// AddRow adds a row to the specified table in JAM.AI.
func AddRow(client *http.Client, tableType models.TableType, tableId string, messages map[string]string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/rows/add", BASE_URL, tableType)
	data := models.AddRowRequest{
		TableID: tableId,
		Data:    []map[string]string{messages},
		Stream:  true,
	}

	resp, err := sendRequest(client, "POST", url, data)
	if err != nil {
		log.Printf("Error generating text during interaction: %v", err)
		return nil, fmt.Errorf("error marshaling data: %w", err)
	}

	return resp, nil
}

// NOTE: THESE COMMENTED FUNCTIONS ARE NOT TESTED, ITS JUST A ROUGH IMPLEMENTATION!

// func getChatTableRows(client *http.Client, tableId string, offset, limit int) {
//     url := fmt.Sprintf("%s/chat/%s/rows?offset=%d&limit=%d", BASE_URL, tableId, offset, limit)

//     req, err := http.NewRequest("GET", url, nil)
//     if err != nil {
//         log.Printf("Error creating request: %v", err)
//         return
//     }

//     req.Header.Set("Accept", "application/json")

//     resp, err := client.Do(req)
//     if err != nil {
//         log.Printf("Error getting chat table rows: %v", err)
//         return
//     }
//     defer resp.Body.Close()

//     var data interface{}
//     err = json.NewDecoder(resp.Body).Decode(&data)
//     if err != nil {
//         log.Printf("Error decoding response: %v", err)
//         return
//     }
//     log.Println("Chat table rows retrieved successfully:", data)
// }

// func deleteConversationTable(client *http.Client, conversationId string) {
//     url := fmt.Sprintf("%s/chat/%s", BASE_URL, conversationId)

//     req, err := http.NewRequest("DELETE", url, nil)
//     if err != nil {
//         log.Printf("Error creating request: %v", err)
//         return
//     }

//     req.Header.Set("Content-Type", "application/json")

//     resp, err := client.Do(req)
//     if err != nil {
//         log.Printf("Error deleting conversation table: %v", err)
//         return
//     }
//     defer resp.Body.Close()

//     log.Println("Conversation table deleted successfully.")
// }
