package lib

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	publish     = "publish"
	subscribe   = "subscribe"
	unsubscribe = "unsubscribe"
)

// Pubsub structure for storing list of the clients
type Pubsub struct {
	Clients       []Client
	Subscriptions []Subscription
}

// Client structure for representing a client informations
type Client struct {
	ID   string
	Conn *websocket.Conn
}

// Subscription structure denotes single subscription
type Subscription struct {
	Topic  string
	Client *Client
}

// Message structure is for storing incoming messages
type Message struct {
	Action  string          `json:"action"`
	Topic   string          `json:"topic"`
	Message json.RawMessage `json:"message"`
}

func (ps *Pubsub) subscribe(client *Client, topic string) *Pubsub {
	clientSub := ps.getSubscriptions(topic, client)
	if len(clientSub) > 0 {
		// Means client is subscribed before to this topic
		return ps
	}
	newSubscription := Subscription{
		Topic:  topic,
		Client: client,
	}
	ps.Subscriptions = append(ps.Subscriptions, newSubscription)
	return ps
}

func (ps *Pubsub) publish(topic string, message []byte, excludedClient *Client) {
	subscriptions := ps.getSubscriptions(topic, nil)
	for _, subscription := range subscriptions {
		log.Println("Sending to client: ", subscription.Client.ID)
		subscription.Client.Conn.WriteMessage(1, message)
		subscription.Client.send(message)
	}
}

func (client *Client) send(message []byte) error {
	return client.Conn.WriteMessage(1, message)
}

// unsubscribe function for removing subscription from the pubsub
func (ps *Pubsub) unsubscribe(client *Client, topic string) *Pubsub {
	for index, sub := range ps.Subscriptions {
		if sub.Client.ID == client.ID && sub.Topic == topic {
			// Found the subscription from client
			ps.Subscriptions = append(ps.Subscriptions[:index], ps.Subscriptions[index+1:]...)
		}
	}
	return ps
}

// RemoveClient function for removing the clients from the socket
func (ps *Pubsub) RemoveClient(client Client) *Pubsub {
	// First remove all the subscription of this client
	for index, sub := range ps.Subscriptions {
		if client.ID == sub.Client.ID {
			ps.Subscriptions = append(ps.Subscriptions[:index], ps.Subscriptions[index+1:]...)
		}
	}
	// Second remove all the client from the list
	for index, c := range ps.Clients {
		if c.ID == client.ID {
			ps.Clients = append(ps.Clients[:index], ps.Clients[index+1:]...)
		}
	}
	return ps
}

func (ps *Pubsub) getSubscriptions(topic string, client *Client) []Subscription {
	var subscriptionList []Subscription
	for _, subscription := range ps.Subscriptions {
		if client != nil {
			if subscription.Client.ID == client.ID && subscription.Topic == topic {
				subscriptionList = append(subscriptionList, subscription)
			}
		} else {
			if subscription.Topic == topic {
				subscriptionList = append(subscriptionList, subscription)
			}
		}
	}
	return subscriptionList
}

// AddClient function for adding new client to the list
func (ps *Pubsub) AddClient(client Client) *Pubsub {
	ps.Clients = append(ps.Clients, client)
	log.Info("Adding a new client to the list: ", client.ID)
	payload := []byte("Hello Client ID" + client.ID)
	client.Conn.WriteMessage(1, payload)
	return ps
}

// HandleRecievedMessage for hendling the recieved message from the server.
func (ps *Pubsub) HandleRecievedMessage(client Client, messageType int, payload []byte) *Pubsub {
	m := Message{}
	err := json.Unmarshal(payload, &m)
	if err != nil {
		log.Println("Something is wrong with the message!")
		return ps
	}
	switch m.Action {
	case publish:
		ps.publish(m.Topic, m.Message, nil)
		break
	case subscribe:
		ps.subscribe(&client, m.Topic)
		log.Println("New subscriber to topic: ", m.Topic, len(ps.Subscriptions))
		break
	case unsubscribe:
		ps.unsubscribe(&client, m.Topic)
		break
	default:
		break
	}
	return ps
}
