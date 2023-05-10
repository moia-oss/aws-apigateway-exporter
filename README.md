# aws-apigateway-exporter

Prometheus Exporter for details of AWS API Gateway metrics that are not available through CloudWatch but are
available via the API. Currently, implemented are metrics about the client certificates and usage plans, especially
the `created_date` and `expiry_date` of each certificate.

The following metrics are then made available with the prefix `aws_apigateway_exporter_`.
All metrics are gauge values:

* `expiration_date` with dimensions `client_certificate_id` and `api_gateway_name`
* `created_date` with dimensions `client_certificate_id` and `api_gateway_name`
* `used_quota` with dimensions `usage_plan_id`, `usage_plan_name`, and `usage_key_id`
* `remaining_quota` with dimensions `usage_plan_id`, `usage_plan_name`, and `usage_key_id`
* `quota_limit` with dimensions `usage_plan_id`, and `usage_plan_name`

## Building and running

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

Make sure your machine has Go 1.15 installed, then run `make docker-build` to build the service as well 
as the docker image. You can also just use the provided docker images:
https://gallery.ecr.aws/moia-oss/aws-apigateway-exporter

For running the latest provided image:
`docker run -p 9389:9389 public.ecr.aws/moia-oss/aws-apigateway-exporter:latest`

For running a local image:
`docker run -p 9389:9389 moia/aws-apigateway-exporter`.

Then Prometheus can scrape port 9389 for these metrics.

## Versioning on Public ECR

With version `0.6.2` this project switches from Dockerhub to AWS ECR Public Gallery, in order to
work around the limits from Dockerhub and because we assume that this is mainly used in AWS environments.

Every commit on `main` gets pushed as a new image with the `latest` tag.

We recommend to use tagged versions in production.

You can find the available tags on [AWS ECR Public Gallery](https://gallery.ecr.aws/moia-oss/aws-apigateway-exporter).

### Pushing a versioned image

In order to tag a new version do the following:

```shell script
git tag 0.[n] -m "your message"
# This will only work if your tag includes a message
git push --follow-tags
```
