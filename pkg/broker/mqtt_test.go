package broker

import (
	"errors"
	"os"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockToken mocks the mqtt.Token interface
type MockToken struct {
	mock.Mock
	mqtt.Token
}

func (m *MockToken) Wait() bool { return true }
func (m *MockToken) Error() error {
	args := m.Called()
	return args.Error(0)
}

// MockClient mocks the mqtt.Client interface
type MockClient struct {
	mock.Mock
	mqtt.Client
}

func (m *MockClient) Connect() mqtt.Token     { return m.Called().Get(0).(mqtt.Token) }
func (m *MockClient) Disconnect(quiesce uint) { m.Called(quiesce) }
func (m *MockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return m.Called(topic, qos, retained, payload).Get(0).(mqtt.Token)
}
func (m *MockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	args := m.Called(topic, qos, mock.Anything)
	return args.Get(0).(mqtt.Token)
}

func TestNewMQTTClient(t *testing.T) {
	t.Run("Fails_When_No_Broker", func(t *testing.T) {
		os.Setenv("MQTT_BROKER", "tcp://localhost:12345") // Non-existent port
		client, err := NewMQTTClient("clientId")

		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("Fails_When_No_Env", func(t *testing.T) {
		os.Unsetenv("MQTT_BROKER")
		client, err := NewMQTTClient("clientId")

		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "not defined")
	})
}

func TestPublish(t *testing.T) {
	mockClient := new(MockClient)
	mockToken := new(MockToken)

	// Setup expectations
	topic := "sensors/data"
	payload := []byte("hello")

	mockToken.On("Error").Return(nil)
	mockClient.On("Publish", topic, byte(0), false, payload).Return(mockToken)

	// Wrap the mock
	c := &MQTTClient{mc: mockClient}

	err := c.Publish(topic, payload)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPublish_Error(t *testing.T) {
	mockClient := new(MockClient)
	mockToken := new(MockToken)

	mockToken.On("Error").Return(errors.New("network failure"))
	mockClient.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockToken)

	c := &MQTTClient{mc: mockClient}
	err := c.Publish("topic", []byte("data"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network failure")
}

func TestSubscribe(t *testing.T) {
	t.Run("Successful_Subscription", func(t *testing.T) {
		mockClient := new(MockClient)
		mockToken := new(MockToken)

		topic := "sensors/data"
		mockToken.On("Error").Return(nil)
		mockClient.On("Subscribe", topic, byte(0), mock.Anything).Return(mockToken)

		c := &MQTTClient{mc: mockClient}

		handler := func(client mqtt.Client, msg mqtt.Message) {}

		err := c.Subscribe(topic, handler)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)

	})

	t.Run("Subscription Failure", func(t *testing.T) {
		mockClient := new(MockClient)
		mockToken := new(MockToken)

		mockClient.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(mockToken)

		os.Setenv("MQTT_BROKER", "tcp://localhost:9999")
		c, _ := NewMQTTClient("clientId")
		err := c.Subscribe("invalid/topic", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot be nil")
	})
}

func TestClose(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("Disconnect", uint(250)).Return()

	c := &MQTTClient{mc: mockClient}
	err := c.Close()

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
