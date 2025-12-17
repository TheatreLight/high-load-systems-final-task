package utils

import (
	"encoding/json"
	"log"
	"time"
)

// AuditLog represents a structured audit log entry
type AuditLog struct {
	Action    string    `json:"action"`
	UserID    int       `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// LogUserAction logs user-related actions asynchronously with JSON structure
// This function should be called with 'go' keyword for async execution
func LogUserAction(action string, userID int) {
	logEntry := AuditLog{
		Action:    action,
		UserID:    userID,
		Timestamp: time.Now(),
	}

	logData, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Error marshaling audit log: %v", err)
		return
	}

	log.Printf("AUDIT: %s", string(logData))
}

// SendNotification simulates sending a notification asynchronously
// This function should be called with 'go' keyword for async execution
func SendNotification(userID int, message string) {
	notification := map[string]interface{}{
		"user_id":   userID,
		"message":   message,
		"timestamp": time.Now(),
	}

	notificationData, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling notification: %v", err)
		return
	}

	log.Printf("NOTIFICATION: %s", string(notificationData))
}

// HandleError logs errors asynchronously with context
func HandleError(err error, context string) {
	if err != nil {
		errorLog := map[string]interface{}{
			"error":     err.Error(),
			"context":   context,
			"timestamp": time.Now(),
		}

		errorData, marshalErr := json.Marshal(errorLog)
		if marshalErr != nil {
			log.Printf("Error marshaling error log: %v", marshalErr)
			return
		}

		log.Printf("ERROR: %s", string(errorData))
	}
}

// LogInfo logs an informational message
func LogInfo(message string) {
	log.Printf("INFO: %s [%s]", message, time.Now().Format(time.RFC3339))
}

// LogError logs an error with context
func LogError(err error, context string) {
	if err != nil {
		log.Printf("ERROR [%s]: %v - %s", context, err, time.Now().Format(time.RFC3339))
	}
}

// LogWarning logs a warning message
func LogWarning(message string) {
	log.Printf("WARNING: %s [%s]", message, time.Now().Format(time.RFC3339))
}
