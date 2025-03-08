#!/bin/sh
set -e

echo "Starting AWS resources initialization..."

# Wait for LocalStack to be ready
echo "Waiting for LocalStack to be fully ready..."
sleep 5

# Create S3 bucket for metrics
echo "Creating S3 bucket for metrics storage..."
aws --endpoint-url=http://localstack:4566 s3 mb s3://grades-metrics
aws --endpoint-url=http://localstack:4566 s3api put-bucket-acl --bucket grades-metrics --acl private

# Setup CloudWatch Logs group
echo "Setting up CloudWatch Logs..."
aws --endpoint-url=http://localstack:4566 logs create-log-group --log-group-name /grades-service
aws --endpoint-url=http://localstack:4566 logs create-log-stream --log-group-name /grades-service --log-stream-name application

# Create IAM role and policy for the application
echo "Setting up IAM roles and policies..."
aws --endpoint-url=http://localstack:4566 iam create-role \
  --role-name grades-app-role \
  --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ecs-tasks.amazonaws.com"},"Action":"sts:AssumeRole"}]}'

aws --endpoint-url=http://localstack:4566 iam create-policy \
  --policy-name grades-app-policy \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudwatch:PutMetricData",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "s3:GetObject",
          "s3:PutObject",
          "s3:ListBucket"
        ],
        "Resource": "*"
      }
    ]
  }'

aws --endpoint-url=http://localstack:4566 iam attach-role-policy \
  --role-name grades-app-role \
  --policy-arn arn:aws:iam::000000000000:policy/grades-app-policy

echo "AWS resources initialization completed!"