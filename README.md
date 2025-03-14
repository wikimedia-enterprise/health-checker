# health-checker

A Go package for performing health checks on HTTP endpoints, AWS S3, AWS Cognito and Apache Kafka. Designed to integrate seamlessly with applications needing periodic or on-demand service health monitoring.

## Features
- **HTTP Health Checks**: Validate the availability of web services.
- **S3 Health Checks**: Ensure AWS S3 bucket accessibility.
- **Cognito Health Checks**: Ensure AWS Cognito accessibility.
- **Kafka Health Checks**: Monitor Kafka producer/consumer activity and offsets.
- **Asynchronous Health Checks**: Run periodic checks without blocking the main application.
- **Integration with `github.com/hellofresh/health-go`** for easy observability.

## Installation

To install the package, run:

```sh
 go get github.com/wikimedia-enterprise/health-checker
```

## Usage

### Basic HTTP Health Check

```go
package main

import (
	"log"
	"net/http"
	"time"

	health "github.com/wikimedia-enterprise/health-checker/health"
)

func main() {
	httpChecker := health.NewHTTPChecker(health.HTTPCheckerConfig{
		URL:            "https://www.wikipedia.org",
		Name:           "wikipedia-http-check",
		ExpectedStatus: http.StatusOK,
	})

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, nil, 1, httpChecker)
	if err != nil {
		log.Fatalf("Failed to setup health checks: %v", err)
	}

	health.StartHealthCheckServer(h, ":8080")
	log.Println("Health check server listening on :8080")

	select {}
}
```

### S3 Health Check

```go
package main

import (
	"log"
	"time"

	health "github.com/wikimedia-enterprise/health-checker/health"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session: %v", err)
	}

	s3Client := s3.New(sess)

	s3Checker, err := health.NewS3Checker(health.S3CheckerConfig{
		BucketName: "my-bucket",
		Name:       "s3-bucket-check",
		S3Client:   s3Client,
	})
	if err != nil {
		log.Fatalf("Failed to create S3 checker: %v", err)
	}

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, nil, 1, s3Checker)
	if err != nil {
		log.Fatalf("Failed to setup health checks: %v", err)
	}

	health.StartHealthCheckServer(h, ":8080")
	log.Println("Health check server listening on :8080")

	select {}
}
```

### Kafka Health Check (Async)

```go
package main

import (
	"log"
	"time"
	"context"

	health "github.com/wikimedia-enterprise/health-checker/health"
)

func main() {
	sync, err := health.NewSyncKafkaChecker(health.SyncKafkaChecker{
		Name:           "kafka-health-check",
		Interval:       10 * time.Second,
		Producer:       nil, // Provide a valid producer client
		Consumer:       nil, // Provide a valid consumer client
		ConsumerTopics: []string{"test-topic-1"},
		ProducerTopics: []string{"test-topic-2"},
		MaxLag:         1000,
	}, health.NewConsumerOffsetStore())
	if err != nil {
		log.Fatalf("Failed to set up Kafka sync checks: %v", err)
	}
	kafkaChecker := health.NewAsyncKafkaChecker(sync)

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, nil, 1, kafkaChecker)
	if err != nil {
		log.Fatalf("Failed to set up health checks: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaChecker.Start(ctx)

	health.StartHealthCheckServer(h, ":8080")
	log.Println("Health check server listening on :8080")

	select {}
}
```

### Combined Health Check (HTTP, S3, Kafka)

```go
package main

import (
	"log"
	"net/http"
	"time"
	"context"

	health "github.com/wikimedia-enterprise/health-checker/health"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	s3Client := s3.New(sess)

	httpChecker := health.NewHTTPChecker(health.HTTPCheckerConfig{
		URL: "https://www.wikipedia.org", Name: "wikipedia-http-check", ExpectedStatus: http.StatusOK,
	})

	s3Checker, _ := health.NewS3Checker(health.S3CheckerConfig{
		BucketName: "my-bucket", Name: "s3-bucket-check", S3Client: s3Client,
	})

	sync, err := health.NewSyncKafkaChecker(health.SyncKafkaChecker{
		Name:           "kafka-health-check",
		Interval:       10 * time.Second,
		Producer:       nil, // Provide a valid producer client
		Consumer:       nil, // Provide a valid consumer client
		ConsumerTopics: []string{"test-topic-1"},
		ProducerTopics: []string{"test-topic-2"},
		MaxLag:         1000,
	}, health.NewConsumerOffsetStore())
	if err != nil {
		log.Fatalf("Failed to set up Kafka sync checks: %v", err)
	}
	kafkaChecker := health.NewAsyncKafkaChecker(sync)

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, nil, 1, httpChecker, s3Checker, kafkaChecker)
	if err != nil {
		log.Fatalf("Failed to setup health checks: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaChecker.Start(ctx)

	health.StartHealthCheckServer(h, ":8080")
	log.Println("Health check server listening on :8080")

	select {}
}
```

### Report Card

![Build](https://github.com/wikimedia-enterprise/health-checker/actions/workflows/go.yml/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/wikimedia-enterprise/health-checker)
![GoDoc](https://pkg.go.dev/badge/github.com/wikimedia-enterprise/health-checker)


## License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.
