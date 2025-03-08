package aws

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// LogClient interface for sending logs to CloudWatch Logs
type LogClient interface {
	SendLog(message string, level string, metadata map[string]string) error
}

// CloudWatchLogger implements LogClient for AWS CloudWatch Logs
type CloudWatchLogger struct {
	svc           *cloudwatchlogs.CloudWatchLogs
	logGroupName  string
	logStreamName string
	sequenceToken *string
}

// NewCloudWatchLogger creates a new CloudWatch logs client
func NewCloudWatchLogger() (*CloudWatchLogger, error) {
	endpoint := os.Getenv("AWS_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	logGroupName := os.Getenv("AWS_LOG_GROUP")
	if logGroupName == "" {
		logGroupName = "/grades-service"
	}
	logStreamName := os.Getenv("AWS_LOG_STREAM")
	if logStreamName == "" {
		logStreamName = "application"
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

	// Create CloudWatch Logs client
	svc := cloudwatchlogs.New(sess)

	return &CloudWatchLogger{
		svc:           svc,
		logGroupName:  logGroupName,
		logStreamName: logStreamName,
	}, nil
}

// SendLog sends a log entry to CloudWatch Logs
func (c *CloudWatchLogger) SendLog(message string, level string, metadata map[string]string) error {
	// Create log event data
	logEvent := map[string]interface{}{
		"message":   message,
		"level":     level,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Add metadata if provided
	for k, v := range metadata {
		logEvent[k] = v
	}

	// Convert to JSON
	logData, err := json.Marshal(logEvent)
	if err != nil {
		return errors.Wrap(err, "failed to marshal log event")
	}

	// Create input for PutLogEvents
	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{
				Message:   aws.String(string(logData)),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
			},
		},
	}

	// Add sequence token if we have one
	if c.sequenceToken != nil {
		input.SequenceToken = c.sequenceToken
	}

	// Send logs to CloudWatch
	resp, err := c.svc.PutLogEvents(input)
	if err != nil {
		// If the error is InvalidSequenceTokenException, we need to get the correct token
		if _, ok := err.(*cloudwatchlogs.InvalidSequenceTokenException); ok {
			// Get the current sequence token
			descInput := &cloudwatchlogs.DescribeLogStreamsInput{
				LogGroupName:        aws.String(c.logGroupName),
				LogStreamNamePrefix: aws.String(c.logStreamName),
			}
			descOutput, descErr := c.svc.DescribeLogStreams(descInput)
			if descErr != nil {
				return errors.Wrap(descErr, "failed to describe log streams")
			}

			// Find our log stream
			for _, stream := range descOutput.LogStreams {
				if *stream.LogStreamName == c.logStreamName {
					c.sequenceToken = stream.UploadSequenceToken
					break
				}
			}

			// Retry with the new sequence token
			if c.sequenceToken != nil {
				input.SequenceToken = c.sequenceToken
				resp, err = c.svc.PutLogEvents(input)
				if err != nil {
					return errors.Wrap(err, "failed to put log events after token refresh")
				}
			}
		} else {
			return errors.Wrap(err, "failed to put log events")
		}
	}

	// Update sequence token for next call
	if resp != nil && resp.NextSequenceToken != nil {
		c.sequenceToken = resp.NextSequenceToken
	}

	return nil
}

// BatchLogger accumulates logs and sends them in batches
type BatchLogger struct {
	client       LogClient
	batchSize    int
	flushTimeout time.Duration
	logChan      chan logEntry
	done         chan struct{}
}

type logEntry struct {
	message  string
	level    string
	metadata map[string]string
}

// NewBatchLogger creates a new batch logger
func NewBatchLogger(client LogClient, batchSize int, flushInterval time.Duration) *BatchLogger {
	logger := &BatchLogger{
		client:       client,
		batchSize:    batchSize,
		flushTimeout: flushInterval,
		logChan:      make(chan logEntry, batchSize*2),
		done:         make(chan struct{}),
	}

	go logger.processLogs()

	return logger
}

// processLogs processes logs in batches
func (b *BatchLogger) processLogs() {
	var buffer []logEntry
	ticker := time.NewTicker(b.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case entry := <-b.logChan:
			buffer = append(buffer, entry)
			if len(buffer) >= b.batchSize {
				b.flushLogs(buffer)
				buffer = make([]logEntry, 0, b.batchSize)
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				b.flushLogs(buffer)
				buffer = make([]logEntry, 0, b.batchSize)
			}
		case <-b.done:
			// Flush any remaining logs
			if len(buffer) > 0 {
				b.flushLogs(buffer)
			}
			return
		}
	}
}

// flushLogs sends the buffered logs to CloudWatch
func (b *BatchLogger) flushLogs(logs []logEntry) {
	for _, entry := range logs {
		if err := b.client.SendLog(entry.message, entry.level, entry.metadata); err != nil {
			log.Error().Err(err).Msg("Failed to send log to CloudWatch")
		}
	}
}

// Log sends a log entry to the batch processor
func (b *BatchLogger) Log(message string, level string, metadata map[string]string) {
	b.logChan <- logEntry{
		message:  message,
		level:    level,
		metadata: metadata,
	}
}

// Close stops the batch logger
func (b *BatchLogger) Close() {
	close(b.done)
}
