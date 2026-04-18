package messaging

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// AWSSNSProvider implementa MessageProvider para SMS usando AWS SNS.
type AWSSNSProvider struct {
	client *sns.Client
}

// NewAWSSNSProvider crea un nuevo proveedor de SNS.
func NewAWSSNSProvider(ctx context.Context) (*AWSSNSProvider, error) {
	// Carga la configuración por defecto de AWS (variables de entorno o config file)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error cargando configuración de AWS: %w", err)
	}

	client := sns.NewFromConfig(cfg)
	return &AWSSNSProvider{
		client: client,
	}, nil
}

func (p *AWSSNSProvider) Send(ctx context.Context, to string, content string) error {
	// En AWS SNS, para enviar SMS directo se usa el número de teléfono como destino
	input := &sns.PublishInput{
		Message:     aws.String(content),
		PhoneNumber: aws.String(to),
	}

	_, err := p.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("error enviando SMS con AWS SNS: %w", err)
	}

	return nil
}

func (p *AWSSNSProvider) ProviderType() string {
	return "SMS"
}
