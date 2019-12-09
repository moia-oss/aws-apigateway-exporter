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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "aws_apigateway_exporter"
)

var (
	// BuildTime represents the time of the build
	BuildTime = "N/A"
	// Version represents the Build SHA-1 of the binary
	Version = "N/A"
)

// Exporter collects metrics for API Gateway client certificates.
type Exporter struct {
	apigateway     *apigateway.APIGateway
	expirationDate *prometheus.Desc
	createdDate    *prometheus.Desc
	up             *prometheus.Desc
	usedQuota      *prometheus.Desc
	remainingQuota *prometheus.Desc
	quotaLimit     *prometheus.Desc
}

// Describe implements prometheus.Collector interface
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.expirationDate
	ch <- e.createdDate
	ch <- e.up
	ch <- e.usedQuota
	ch <- e.remainingQuota
	ch <- e.quotaLimit
}

// Collect implements prometheus.Collector interface
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	up := 1 // indicates any error while scraping

	err := e.collectUsageMetrics(&up, ch)
	if err != nil {
		up = 0
		log.Error("Failed to get usage plans")
	}

	err = e.collectCertificateMetrics(&up, ch)

	if err != nil {
		up = 0
		log.Errorf("Failed to get api gateways %s", err)
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, float64(up))
}

func (e *Exporter) collectCertificateMetrics(up *int, ch chan<- prometheus.Metric) error {
	return e.apigateway.GetRestApisPages(&apigateway.GetRestApisInput{},
		func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
			for _, restAPI := range page.Items {
				stagesResponse, stageErr := e.apigateway.GetStages(&apigateway.GetStagesInput{RestApiId: restAPI.Id})
				if stageErr != nil {
					*up = 0
					log.Errorf("Failed to get stages for API Gateway %s: %s", *restAPI.Id, stageErr)
					continue
				}

				for _, stage := range stagesResponse.Item {
					if stage.ClientCertificateId != nil {
						cert, err := e.apigateway.GetClientCertificate(&apigateway.GetClientCertificateInput{ClientCertificateId: stage.ClientCertificateId})
						if err != nil {
							*up = 0
							log.Errorf("Failed to get client certificates %s for API Gateway %s: %s", *stage.ClientCertificateId, *restAPI.Id, err)
							continue
						}
						ch <- prometheus.MustNewConstMetric(e.expirationDate, prometheus.GaugeValue, float64(cert.ExpirationDate.Unix()), *cert.ClientCertificateId, *restAPI.Name)
						ch <- prometheus.MustNewConstMetric(e.createdDate, prometheus.GaugeValue, float64(cert.CreatedDate.Unix()), *cert.ClientCertificateId, *restAPI.Name)
					}
				}
			}
			return lastPage
		})
}

func (e *Exporter) collectUsageMetrics(up *int, ch chan<- prometheus.Metric) error {
	log.Info("collecting Usage Metrics")
	return e.apigateway.GetUsagePlansPages(&apigateway.GetUsagePlansInput{},
		func(page *apigateway.GetUsagePlansOutput, lastPage bool) bool {
			for _, plan := range page.Items {
				today := aws.String(time.Now().Format("2006-01-02"))
				usage, usageErr := e.apigateway.GetUsage(&apigateway.GetUsageInput{
					EndDate:     today,
					StartDate:   today,
					UsagePlanId: plan.Id,
				})
				if usageErr != nil {
					*up = 0
					log.Errorf("Failed to get usage data for API Usage Plan %s (%s): %s", *plan.Id, *plan.Name, usageErr)
					continue
				}

				for key, val := range usage.Items {
					ch <- prometheus.MustNewConstMetric(
						e.usedQuota,
						prometheus.GaugeValue,
						// note that we always get the first element of val because we only ask for one day of data
						float64(*val[0][0]),
						*plan.Id,
						*plan.Name,
						key,
					)
					ch <- prometheus.MustNewConstMetric(
						e.remainingQuota,
						prometheus.GaugeValue,
						// note that we always get the first element of val because we only ask for one day of data
						float64(*val[0][1]),
						*plan.Id,
						*plan.Name,
						key,
					)
				}

				if plan.Quota != nil {
					ch <- prometheus.MustNewConstMetric(
						e.quotaLimit,
						prometheus.GaugeValue,
						float64(*plan.Quota.Limit),
						*plan.Id,
						*plan.Name,
					)
				}
			}

			return lastPage
		})
}

func registerSignals() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Received SIGTERM, exiting...")
		os.Exit(1)
	}()
}

func main() {
	var (
		showVersion = kingpin.Flag("version", "Print version information").Bool()
		listenAddr  = kingpin.Flag("listen-address", "The address to listen on for HTTP requests.").Default(":9389").String()
		region      = kingpin.Flag("region", "The AWS region to use.").Default("eu-central-1").String()
	)
	registerSignals()
	kingpin.Parse()

	if *showVersion {
		fmt.Printf("Build Time:   %s\n", BuildTime)
		fmt.Printf("Build SHA-1:  %s\n", Version)
		fmt.Printf("Go Version:   %s\n", runtime.Version())
		os.Exit(0)
	}

	log.Infof("Starting `aws-apigateway-exporter`: Build Time: '%s' Build SHA-1: '%s'\n", BuildTime, Version)

	stsSession := session.Must(session.NewSession(&aws.Config{Region: region}))
	exporter := createExporter(stsSession, region)
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
			log.Errorf("Error on writing default response %s", err)
		}
	})
	log.Info("Listening on", *listenAddr)
	err := http.ListenAndServe(*listenAddr, mux)
	if err != nil {
		log.Errorf("Error on serving the requests %s", err)
	}
}

func createExporter(stsSession client.ConfigProvider, region *string) *Exporter {
	return &Exporter{
		apigateway: apigateway.New(stsSession),
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
