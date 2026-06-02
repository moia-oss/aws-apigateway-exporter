package main

/*
   Copyright 2019 MOIA GmbH

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "aws_apigateway_exporter"
)

var (
	// BuildTime represents the time of the build.
	BuildTime = "N/A"
	// Version represents the Build SHA-1 of the binary.
	Version = "N/A"

	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

// Exporter collects metrics for API Gateway client certificates.
type Exporter struct {
	apigateway     *apigateway.Client
	expirationDate *prometheus.Desc
	createdDate    *prometheus.Desc
	up             *prometheus.Desc
	usedQuota      *prometheus.Desc
	remainingQuota *prometheus.Desc
	quotaLimit     *prometheus.Desc
}

// Describe implements prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.expirationDate
	ch <- e.createdDate
	ch <- e.up
	ch <- e.usedQuota
	ch <- e.remainingQuota
	ch <- e.quotaLimit
}

// Collect implements prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	up := 1 // indicates any error while scraping

	err := e.collectUsageMetrics(&up, ch)
	if err != nil {
		up = 0
		sugar.Error("Failed to get usage plans")
	}

	err = e.collectCertificateMetrics(&up, ch)
	if err != nil {
		up = 0
		sugar.Errorf("Failed to get api gateways %s", err)
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, float64(up))
}

func (e *Exporter) collectCertificateMetrics(up *int, ch chan<- prometheus.Metric) error {
	ctx := context.Background()
	paginator := apigateway.NewGetRestApisPaginator(e.apigateway, &apigateway.GetRestApisInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("get API Gateway REST APIs: %w", err)
		}

		for _, restAPI := range page.Items {
			apiID := aws.ToString(restAPI.Id)
			if apiID == "" {
				*up = 0
				sugar.Warn("Skipping API Gateway without an ID")
				continue
			}
			apiName := aws.ToString(restAPI.Name)

			stagesResponse, stageErr := e.apigateway.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: aws.String(apiID)})
			if stageErr != nil {
				*up = 0
				sugar.Errorf("Failed to get stages for API Gateway %s: %s", apiID, stageErr)
				continue
			}

			for _, stage := range stagesResponse.Item {
				if stage.ClientCertificateId != nil {
					cert, err := e.apigateway.GetClientCertificate(ctx, &apigateway.GetClientCertificateInput{ClientCertificateId: stage.ClientCertificateId})
					if err != nil {
						*up = 0
						sugar.Errorf("Failed to get client certificates %s for API Gateway %s: %s", *stage.ClientCertificateId, apiID, err)
						continue
					}
					certID := aws.ToString(cert.ClientCertificateId)
					if certID == "" || cert.ExpirationDate == nil || cert.CreatedDate == nil {
						*up = 0
						sugar.Warnf("Skipping incomplete client certificate %s for API Gateway %s", aws.ToString(stage.ClientCertificateId), apiID)
						continue
					}
					ch <- prometheus.MustNewConstMetric(e.expirationDate, prometheus.GaugeValue, float64(cert.ExpirationDate.Unix()), certID, apiName)
					ch <- prometheus.MustNewConstMetric(e.createdDate, prometheus.GaugeValue, float64(cert.CreatedDate.Unix()), certID, apiName)
				}
			}
		}
	}
	return nil
}

func (e *Exporter) collectUsageMetrics(up *int, ch chan<- prometheus.Metric) error {
	sugar.Info("collecting Usage Metrics")
	ctx := context.Background()
	paginator := apigateway.NewGetUsagePlansPaginator(e.apigateway, &apigateway.GetUsagePlansInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("get API Gateway usage plans: %w", err)
		}

		for _, plan := range page.Items {
			planID := aws.ToString(plan.Id)
			if planID == "" {
				*up = 0
				sugar.Warn("Skipping usage plan without an ID")
				continue
			}
			planName := aws.ToString(plan.Name)
			today := time.Now().Format("2006-01-02")
			usage, usageErr := e.apigateway.GetUsage(ctx, &apigateway.GetUsageInput{
				EndDate:     &today,
				StartDate:   &today,
				UsagePlanId: aws.String(planID),
			})
			if usageErr != nil {
				*up = 0
				sugar.Errorf("Failed to get usage data for API Usage Plan %s (%s): %s", planID, planName, usageErr)
				continue
			}

			for key, val := range usage.Items {
				used, remaining, ok := usageValuesForToday(val)
				if !ok {
					*up = 0
					sugar.Warnf("Skipping unexpected usage data for API Usage Plan %s (%s) and usage key %s", planID, planName, key)
					continue
				}
				ch <- prometheus.MustNewConstMetric(
					e.usedQuota,
					prometheus.GaugeValue,
					float64(used),
					planID,
					planName,
					key,
				)
				ch <- prometheus.MustNewConstMetric(
					e.remainingQuota,
					prometheus.GaugeValue,
					float64(remaining),
					planID,
					planName,
					key,
				)
			}

			if plan.Quota != nil {
				ch <- prometheus.MustNewConstMetric(
					e.quotaLimit,
					prometheus.GaugeValue,
					float64(plan.Quota.Limit),
					planID,
					planName,
				)
			}
		}
	}

	return nil
}

func usageValuesForToday(values [][]int64) (used, remaining int64, ok bool) {
	if len(values) == 0 || len(values[0]) < 2 {
		return 0, 0, false
	}
	return values[0][0], values[0][1], true
}

func registerSignals() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sugar.Info("Received SIGTERM, exiting...")
		os.Exit(1)
	}()
}

func main() {
	var (
		showVersion = kingpin.Flag("version", "Print version information").Bool()
		listenAddr  = kingpin.Flag("listen-address", "The address to listen on for HTTP requests.").Default(":9389").String()
		region      = kingpin.Flag("region", "The AWS region to use.").Default("eu-central-1").String()
	)

	logger, _ = zap.NewProduction()
	// nolint: errcheck
	defer logger.Sync()
	sugar = logger.Sugar()

	registerSignals()
	kingpin.Parse()

	sugar.Info("Starting...")

	if *showVersion {
		fmt.Printf("Build Time:   %s\n", BuildTime)
		fmt.Printf("Build SHA-1:  %s\n", Version)
		fmt.Printf("Go Version:   %s\n", runtime.Version())
		os.Exit(0)
	}

	sugar.Infof("Starting `aws-apigateway-exporter`: Build Time: '%s' Build SHA-1: '%s'\n", BuildTime, Version)

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(*region))
	if err != nil {
		sugar.Fatalf("Failed to load AWS config: %v", err)
	}
	exporter := createExporter(cfg, region)
	prometheus.MustRegister(exporter)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(
			`<html>
				<head>
					<title>AWS API Gateway Metrics Exporter</title>
				</head>
             	<body>
             		<h1>AWS API Gateway Metrics Exporter</h1>
             		<p><a href='/metrics'>Metrics</a></p>
             	</body>
             </html>`))
		if err != nil {
			sugar.Errorf("Error on writing default response %s", err)
		}
	})
	sugar.Info("Listening on", *listenAddr)
	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		sugar.Errorf("Error on serving the requests %s", err)
	}
}

func createExporter(cfg aws.Config, region *string) *Exporter {
	return &Exporter{
		apigateway: apigateway.NewFromConfig(cfg),
		expirationDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "expiration_date"),
			"The expiration date of the client certificate as Unix timestamp.",
			[]string{"client_certificate_id", "api_gateway_name"},
			map[string]string{"region": *region},
		),
		createdDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "created_date"),
			"The creation date of the client certificate as Unix timestamp.",
			[]string{"client_certificate_id", "api_gateway_name"},
			map[string]string{"region": *region},
		),
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Indicates a successful scrape.",
			nil,
			nil,
		),
		usedQuota: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "used_quota"),
			"The daily logs of the used quota.",
			[]string{"usage_plan_id", "usage_plan_name", "usage_key_id"},
			map[string]string{"region": *region},
		),
		remainingQuota: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "remaining_quota"),
			"The daily logs of the remaining quota.",
			[]string{"usage_plan_id", "usage_plan_name", "usage_key_id"},
			map[string]string{"region": *region},
		),
		quotaLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "quota_limit"),
			"The limit of the plan for a specific time period.",
			[]string{"usage_plan_id", "usage_plan_name"},
			map[string]string{"region": *region},
		),
	}
}
