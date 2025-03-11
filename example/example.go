package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	healthchecks "github.com/wikimedia-enterprise/health-checker/health"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

func NewS3(env *dummyEnv) s3iface.S3API {
	cfg := &aws.Config{
		Region: aws.String(env.AWSRegion),
	}

	if len(env.AWSID) > 0 && len(env.AWSKey) > 0 {
		cfg.Credentials = credentials.NewStaticCredentials(env.AWSID, env.AWSKey, "")
	}

	if len(env.AWSURL) > 0 {
		cfg.Endpoint = aws.String(env.AWSURL)
	}

	if strings.HasPrefix(env.AWSURL, "http://") {
		cfg.DisableSSL = aws.Bool(true)
		cfg.S3ForcePathStyle = aws.Bool(true)
	}

	return s3.New(session.Must(session.NewSession(cfg)))
}

type dummyEnv struct {
	AWSRegion string
	AWSID     string
	AWSKey    string
	AWSURL    string
}

func main() {
	httpChecker1 := healthchecks.NewHTTPChecker(healthchecks.HTTPCheckerConfig{
		URL:              "https://www.wikipedia.org",
		Name:             "wikipedia-http-check",
		ExpectedStatuses: []int{http.StatusOK},
	})
	httpChecker2 := healthchecks.NewHTTPChecker(healthchecks.HTTPCheckerConfig{
		URL:              "https://www.wikimedia.org",
		Name:             "wikimedia-http-check",
		ExpectedStatuses: []int{http.StatusOK},
	})

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session: %v", err)
	}

	dummyS3Config := &dummyEnv{
		AWSRegion: *sess.Config.Region,
	}

	s3Client := NewS3(dummyS3Config)

	s3Checker, err := healthchecks.NewS3Checker(healthchecks.S3CheckerConfig{
		BucketName: "your-bucket-name",
		Name:       "s3-bucket-check",
		S3Client:   s3Client,
	})
	if err != nil {
		log.Fatalf("Failed to create S3 checker: %v", err)
	}

	loggerCallback := func(name string, checkType string, result error) {
		if result != nil {
			fmt.Printf("Error from %s: %v\n", name, result)
		}
	}

	h, err := healthchecks.SetupHealthChecks("MyService", "1.0.0", true, loggerCallback, httpChecker1, httpChecker2, s3Checker)
	if err != nil {
		log.Fatalf("Failed to setup health checks: %v", err)
	}

	healthchecks.StartHealthCheckServer(h, ":8080")
	log.Println("Health check server listening on :8080")

	select {}
}
