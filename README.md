# aws-apigateway-exporter

[![CircleCI](https://circleci.com/gh/moia-dev/aws-apigateway-exporter.svg?style=svg&circle-token=dd82a9ad720305dcb160a552f1aab356d13ad46c)](https://circleci.com/gh/moia-dev/aws-apigateway-exporter)

Prometheus Exporter for details of AWS API Gateway metrics that are not available through through CloudWatch but are
available via the API. Currently implemented are metrics about the client certificates and usage plans, especially
the `created_date` and `expiry_date` of each certificate.

The following metrics are then made available with the prefix `aws_apigateway_exporter_`.
All metrics are gauge values:

* `expiration_date` with dimensions `client_certificate_id` and `api_gateway_name`
* `created_date` with dimensions `client_certificate_id` and `api_gateway_name`
* `used_quota` with dimensions `usage_plan_id`, `usage_plan_name`, and `usage_key_id`
* `remaining_quota` with dimensions `usage_plan_id`, `usage_plan_name`, and `usage_key_id`
* `quota_limit` with dimensions `usage_plan_id`, and `usage_plan_name`

## Building and running

Make sure your machine has Go 1.13 installed, then run `make docker-build` to build.

This service also needs the following AWS permissions:

```yaml
Effect: "Allow"
Action:
  - "apigateway:GET"
Resource:
  - "arn:aws:apigateway:eu-central-1::/restapis"
  - "arn:aws:apigateway:eu-central-1::/restapis/*/stages"
  - "arn:aws:apigateway:eu-central-1::/clientcertificates/*"
  - "arn:aws:apigateway:eu-central-1::/usageplans*"
```

`docker run -p 9389:9389 moia/aws-apigateway-exporter` to run. Then Prometheus can scrape port 9389 for these metrics.

## Versioning on Docker Hub

Every commit on `master` gets pushed as a new image with the `latest` tag.

We recommend to use tagged versions in production.

You can find the available tags on [Docker Hub](https://hub.docker.com/repository/docker/moia/aws-apigateway-exporter).

In order to tag a new version do the following:

```shell script
git tag 0.[n] -m "your message"
# This will only work if your tag includes a message
git push --follow-tags
```
