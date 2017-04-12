package notifications

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"bytes"
	"github.com/status-im/status-go/geth/params"
)

// NotificationDeliveryProvider handles the notification delivery
type NotificationDeliveryProvider interface {
	Send(id string, payload string) error
}

// FirebaseProvider represents FCM provider
type FirebaseProvider struct {
	AuthorizationKey       string
	NotificationTriggerURL string
}

// NewFirebaseProvider creates new FCM provider
func NewFirebaseProvider(config *params.FirebaseConfig) *FirebaseProvider {
	return &FirebaseProvider{
		NotificationTriggerURL: config.NotificationTriggerURL,
		AuthorizationKey:       config.AuthorizationKey,
	}
}

// Send triggers sending of Push Notification to a given device id
func (p *FirebaseProvider) Send(id string, payload string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	jsonRequest := strings.Replace(payload, "{{ ID }}", id, 3)
	req, err := http.NewRequest("POST", p.NotificationTriggerURL, bytes.NewBuffer([]byte(jsonRequest)))
	req.Header.Set("Authorization", "key="+p.AuthorizationKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	return nil
}
