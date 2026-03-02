package broker

import (
	"errors"
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
	mockClient := new(MockClient)

	client := NewMQTTClient(mockClient)

	assert.NotNil(t, client, "Constructor should return a valid pointer")
	assert.Equal(t, mockClient, client.mc, "The internal client should match the passed mock")
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
	t.Run("Successful Subscription", func(t *testing.T) {
		mockClient := new(MockClient)
		mockToken := new(MockToken)

		// Define a dummy handler
		handler := func(c mqtt.Client, m mqtt.Message) {}

		// Expectations
		mockToken.On("Error").Return(nil)
		mockClient.On("Subscribe", "sensors/+/data", byte(0), mock.Anything).Return(mockToken)

		// Initialize with your new constructor
		c := NewMQTTClient(mockClient)

		err := c.Subscribe("sensors/+/data", handler)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Subscription Failure", func(t *testing.T) {
		mockClient := new(MockClient)
		mockToken := new(MockToken)

		mockClient.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(mockToken)

		c := NewMQTTClient(mockClient)
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
