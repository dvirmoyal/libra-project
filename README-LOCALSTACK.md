# LocalStack AWS Simulation Environment

This branch contains a development environment that simulates AWS services locally using LocalStack.

## Architecture

```
┌──────────────────┐     ┌───────────────────┐     ┌────────────────────┐
│                  │     │                   │     │                    │
│   Grades API     │────▶│   LocalStack      │────▶│  Simulated AWS     │
│   (Go Service)   │     │   Container       │     │  Services          │
│                  │     │                   │     │                    │
└────────┬─────────┘     └───────────────────┘     └────────────────────┘
         │                                               S3 (Storage)
         │                                               CloudWatch (Metrics)
         │                                               CloudWatch Logs
         │                                               IAM Roles/Policies
         │
         │
         ▼
┌──────────────────┐
│                  │
│   PostgreSQL     │
│   (Simulates     │
│   Aurora/RDS)    │
│                  │
└──────────────────┘
```

## Project Structure

- `/aws-init/setup.sh` - Scripts to initialize AWS resources in LocalStack
- `/internal/aws/metrics.go` - Code for sending metrics to AWS CloudWatch
- `/internal/aws/storage.go` - S3 storage client for file operations
- `/internal/aws/logger.go` - CloudWatch Logs integration for application logs
- `docker-compose-localstack.yaml` - Docker Compose configuration with LocalStack services

## File Descriptions

- **docker-compose-localstack.yaml** - Defines containers for the application, PostgreSQL, LocalStack, and initialization
- **aws-init/setup.sh** - Creates S3 buckets, CloudWatch log groups, and IAM roles when environment starts
- **internal/aws/metrics.go** - Implements a CloudWatch client for application metrics
- **internal/aws/storage.go** - Provides S3 storage functionality for file operations
- **internal/aws/logger.go** - Sends application logs to CloudWatch Logs
- **go.mod (updated)** - Updated dependencies including AWS SDK

## Setup Instructions

1. Create a new branch for LocalStack development:
   ```
   git checkout -b LOCALSTACK
   ```

2. Place all files in their respective directories:
    - Place AWS module files in `internal/aws/`
    - Create `aws-init` directory in project root for setup scripts

3. Start the LocalStack environment:
   ```
   docker-compose -f docker-compose-localstack.yaml up
   ```

4. The environment will simulate:
    - RDS/Aurora PostgreSQL
    - S3 for file storage
    - CloudWatch for metrics
    - CloudWatch Logs for centralized logging
    - IAM for permissions

## Notes

- LocalStack endpoint is available at http://localhost:4566
- All AWS resources are automatically created on startup
- AWS credentials in the local environment are dummy values (test/test)