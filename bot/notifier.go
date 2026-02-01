// Package bot provides a wrapper for the Telegram bot to implement BotNotifier interface
package bot

// Notifier wraps the package-level bot functions to implement services.BotNotifier interface
type Notifier struct{}

// NewNotifier creates a new bot notifier
func NewNotifier() *Notifier {
	return &Notifier{}
}

// SendNotification sends a notification to the admin chat
func (n *Notifier) SendNotification(message string) {
	SendNotification(message)
}

// SendPersonalNotification sends a notification to a specific user
func (n *Notifier) SendPersonalNotification(chatID int64, message string) {
	SendPersonalNotification(chatID, message)
}

// Ensure Notifier implements the BotNotifier interface
var _ interface {
	SendNotification(message string)
	SendPersonalNotification(chatID int64, message string)
} = (*Notifier)(nil)
