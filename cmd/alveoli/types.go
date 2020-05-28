package main

type Device struct {
	Active            bool   `json:"active"`
	Connected         bool   `json:"connected"`
	CreatedAt         int64  `json:"createdAt"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	Password          string `json:"password"`
	ReceivedBytes     int64  `json:"receivedBytes"`
	SentBytes         int64  `json:"sentBytes"`
	SubscriptionCount int    `json:"subscriptionCount"`
}
