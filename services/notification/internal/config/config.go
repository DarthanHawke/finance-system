package config

/* legacy code
type Configuration struct {
	RabbitMQ `mapstructure:",squash"`
}

type RabbitMQ struct {
	URL       string `mapstructure:"RABBITMQ_URL" env-default:"amqp://guest:guest@rabbitmq:5672/"`
	QueueName string `mapstructure:"RABBITMQ_QUEUE" env-default:"transactions"`
}
*/
