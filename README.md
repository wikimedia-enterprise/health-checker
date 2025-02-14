# health-checks

A Go package for performing health checks on HTTP endpoints, AWS S3, and Apache Kafka. Designed to integrate seamlessly with applications needing periodic or on-demand service health monitoring.

## Features
- **HTTP Health Checks**: Validate the availability of web services.
- **S3 Health Checks**: Ensure AWS S3 bucket accessibility.
- **Kafka Health Checks**: Monitor Kafka producer/consumer activity and offsets.
- **Asynchronous Health Checks**: Run periodic checks without blocking the main application.
- **Integration with `github.com/hellofresh/health-go`** for easy observability.

## Installation

To install the package, run:

```sh
 go get github.com/wikimedia-enterprise/health-checks
```

## Usage

### Basic HTTP Health Check

```go
package main

import (
	"log"
	"net/http"
	"time"

	health "github.com/wikimedia-enterprise/health-checks"
)

func main() {
	httpChecker := health.NewHTTPChecker(health.HTTPCheckerConfig{
		URL:            "https://www.wikipedia.org",
		Timeout:        5 * time.Second,
		Name:           "wikipedia-http-check",
		ExpectedStatus: http.StatusOK,
	})

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, httpChecker)
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

	health "github.com/wikimedia-enterprise/health-checks"
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
		Timeout:    5 * time.Second,
		S3Client:   s3Client,
	})
	if err != nil {
		log.Fatalf("Failed to create S3 checker: %v", err)
	}

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, s3Checker)
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

	health "github.com/wikimedia-enterprise/health-checks"
)

func main() {
	kafkaChecker := health.NewAsyncKafkaChecker(health.NewSyncKafkaChecker(health.SyncKafkaChecker{
		Name:           "kafka-health-check",
		Interval:       10 * time.Second,
		Timeout:        5 * time.Second,
		Producer:       nil, // Provide a valid producer client
		Consumer:       nil, // Provide a valid consumer client
		RequiredTopics: []string{"test-topic"},
		MaxLag:         1000,
	}, health.NewConsumerOffsetStore()))

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, kafkaChecker)
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

### Combined Health Check (HTTP, S3, Kafka)

```go
package main

import (
	"log"
	"net/http"
	"time"
	"context"

	health "github.com/wikimedia-enterprise/health-checks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	s3Client := s3.New(sess)

	httpChecker := health.NewHTTPChecker(health.HTTPCheckerConfig{
		URL: "https://www.wikipedia.org", Timeout: 5 * time.Second, Name: "wikipedia-http-check", ExpectedStatus: http.StatusOK,
	})

	s3Checker, _ := health.NewS3Checker(health.S3CheckerConfig{
		BucketName: "my-bucket", Name: "s3-bucket-check", Timeout: 5 * time.Second, S3Client: s3Client,
	})

	kafkaChecker := health.NewAsyncKafkaChecker(health.NewSyncKafkaChecker(health.SyncKafkaChecker{
		Name: "kafka-health-check", Interval: 10 * time.Second, Timeout: 5 * time.Second, RequiredTopics: []string{"test-topic"},
	}, health.NewConsumerOffsetStore()))

	h, err := health.SetupHealthChecks("MyService", "1.0.0", true, httpChecker, s3Checker, kafkaChecker)
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

![Build](https://github.com/wikimedia-enterprise/health-checks/actions/workflows/go.yml/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/wikimedia-enterprise/health-checks)
![GoDoc](https://pkg.go.dev/badge/github.com/wikimedia-enterprise/health-checks)


## License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.
