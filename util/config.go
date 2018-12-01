// Package util provides server configuration and message structure.
package util

type Config struct {
	// UCP username
	User string
	// UCP password
	Password string
	// UCP accesscode
	AccessCode string
	// UCP port
	Port int
	// HTTP address of the web UI
	HttpAddr string
	// Delivery notification delay in milliseconds
	DNDelay int
	// Map of billing identifier to cost
	Tariff map[string]float64
}
