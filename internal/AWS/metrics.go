package aws

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// MetricsClient interface for sending metrics to CloudWatch
type MetricsClient interface {
	PutMetric(name string, value float64, unit string, dimensions map[string]string) error
}

// CloudWatchMetrics implements MetricsClient for AWS CloudWatch
type CloudWatchMetrics struct {
	svc       *cloudwatch.CloudWatch
	namespace string
}

// NewCloudWatchMetrics creates a new CloudWatch metrics client
func NewCloudWatchMetrics() (*CloudWatchMetrics, error) {
	endpoint := os.Getenv("AWS_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	namespace := os.Getenv("AWS_METRICS_NAMESPACE")
	if namespace == "" {
		namespace = "GradesService"
	}

	config := aws.NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		))

	if endpoint != "" {
		config = config.WithEndpoint(endpoint)
	}

	// Create a new AWS session
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}

	// Create CloudWatch client
	svc := cloudwatch.New(sess)

	return &CloudWatchMetrics{
		svc:       svc,
		namespace: namespace,
	}, nil
}

// PutMetric sends a metric to CloudWatch
func (c *CloudWatchMetrics) PutMetric(name string, value float64, unit string, dimensions map[string]string) error {
	// Convert dimensions map to CloudWatch dimensions
	cwDimensions := []*cloudwatch.Dimension{}
	for k, v := range dimensions {
		cwDimensions = append(cwDimensions, &cloudwatch.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	// Create metric data
	metricData := []*cloudwatch.MetricDatum{
		{
			MetricName: aws.String(name),
			Value:      aws.Float64(value),
			Unit:       aws.String(unit),
			Dimensions: cwDimensions,
			Timestamp:  aws.Time(time.Now()),
		},
	}

	// Create input for PutMetricData
	input := &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(c.namespace),
		MetricData: metricData,
	}

	// Send metrics to CloudWatch
	_, err := c.svc.PutMetricData(input)
	if err != nil {
		log.Error().Err(err).
			Str("metric", name).
			Float64("value", value).
			Str("unit", unit).
			Interface("dimensions", dimensions).
			Msg("Failed to put metric to CloudWatch")
		return errors.Wrap(err, "failed to put metric data")
	}

	log.Debug().
		Str("metric", name).
		Float64("value", value).
		Str("unit", unit).
		Interface("dimensions", dimensions).
		Msg("Successfully sent metric to CloudWatch")

	return nil
}

// TimedMetric measures execution time and sends it as a metric
func (c *CloudWatchMetrics) TimedMetric(name string, dimensions map[string]string) func() {
	startTime := time.Now()
	return func() {
		duration := time.Since(startTime)
		err := c.PutMetric(
			name,
			float64(duration.Milliseconds()),
			"Milliseconds",
			dimensions,
		)
		if err != nil {
			log.Error().Err(err).Str("metric", name).Msg("Failed to send timed metric")
		}
	}
}
