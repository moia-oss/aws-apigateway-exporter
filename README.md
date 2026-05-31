# aws-apigateway-exporter

Prometheus Exporter for details of AWS API Gateway metrics that are not available through CloudWatch but are
available via the API. Currently, implemented are metrics about the client certificates and usage plans, especially
the `created_date` and `expiration_date` of each certificate.

The exporter exposes the following gauge metrics. Metrics collected from AWS include a constant `region` label:

* `aws_apigateway_exporter_expiration_date` with labels `client_certificate_id`, `api_gateway_name`, and `region`
* `aws_apigateway_exporter_created_date` with labels `client_certificate_id`, `api_gateway_name`, and `region`
* `aws_apigateway_exporter_used_quota` with labels `usage_plan_id`, `usage_plan_name`, `usage_key_id`, and `region`
* `aws_apigateway_exporter_remaining_quota` with labels `usage_plan_id`, `usage_plan_name`, `usage_key_id`, and `region`
* `aws_apigateway_exporter_quota_limit` with labels `usage_plan_id`, `usage_plan_name`, and `region`
* `aws_apigateway_exporter_up`

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

Replace `eu-central-1` in the policy when running the exporter with another `--region`.

Make sure your machine has Go 1.26 installed, then run `make build-linux` to build the Linux binary. To build
both Linux and Darwin amd64 binaries, run `make build`. To build the container image, run `make docker-build`.
You can also use the provided container images:
https://gallery.ecr.aws/s6w2n1r6/aws-apigateway-exporter

For running the latest provided image:
`docker run -p 9389:9389 public.ecr.aws/s6w2n1r6/aws-apigateway-exporter:latest`

For running a local image built by `make docker-build`:
`docker run -p 9389:9389 moia/aws-apigateway-exporter:$(git describe --always --tags)`.

Then Prometheus can scrape port 9389 for these metrics.

The exporter defaults to `--listen-address=:9389` and `--region=eu-central-1`.

## Versioning on Public ECR

With version `0.6.2` this project switches from Dockerhub to AWS ECR Public Gallery, in order to
work around the limits from Dockerhub and because we assume that this is mainly used in AWS environments.

Every commit on `main` gets pushed as a new image with the `latest` tag.

We recommend to use tagged versions in production.

You can find the available tags on [AWS ECR Public Gallery](https://gallery.ecr.aws/s6w2n1r6/aws-apigateway-exporter).

### Pushing a versioned image

In order to tag a new version do the following:

```shell script
git tag 0.[n] -m "your message"
# This will only work if your tag includes a message
git push --follow-tags
```
