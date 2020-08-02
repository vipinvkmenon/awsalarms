package awsalarms

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/influxdata/telegraf"
	internalaws "github.com/influxdata/telegraf/config/aws"
	"github.com/influxdata/telegraf/filter"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type (
	// CloudWatch contains the configuration and cache for the cloudwatch plugin.
	CloudWatch struct {
		Region         string   `toml:"region"`
		AccessKey      string   `toml:"access_key"`
		SecretKey      string   `toml:"secret_key"`
		RoleARN        string   `toml:"role_arn"`
		Profile        string   `toml:"profile"`
		CredentialPath string   `toml:"shared_credential_file"`
		Token          string   `toml:"token"`
		EndpointURL    string   `toml:"endpoint_url"`
		TagsInclude    []string `toml:"tags_exclude"`
		TagsExclude    []string `toml:"tags_include"`
		RateLimit      int      `toml:"ratelimit"`
		StateValue     string   `toml:"state_value"`

		Log telegraf.Logger `toml:"-"`

		client      cloudwatchClient
		statFilter  filter.Filter
		windowStart time.Time
		windowEnd   time.Time
	}
	cloudwatchClient interface {
		DescribeAlarms(*cloudwatch.DescribeAlarmsInput) (*cloudwatch.DescribeAlarmsOutput, error)
	}
)

// SampleConfig returns the default configuration of the Cloudwatch input plugin.
func (c *CloudWatch) SampleConfig() string {
	return `
  ## Amazon Region
  region = "us-east-1"

  ## Amazon Credentials
  ## Credentials are loaded in the following order
  ## 1) Assumed credentials via STS if role_arn is specified
  ## 2) explicit credentials from 'access_key' and 'secret_key'
  ## 3) shared profile from 'profile'
  ## 4) environment variables
  ## 5) shared credentials file
  ## 6) EC2 Instance Profile
  # access_key = ""
  # secret_key = ""
  # token = ""
  # role_arn = ""
  # profile = ""
  # shared_credential_file = ""

  ## Endpoint to make request against, the correct endpoint is automatically
  ## determined and this option should only be set if you wish to override the
  ## default.
  ##   ex: endpoint_url = "http://localhost:8000"
  # endpoint_url = ""
  # Optional StateValue. Default is set to "ALARAM" 
  #state_value = "ALARM"

`
}

// Description returns a one-sentence description on the Cloudwatch input plugin.
func (c *CloudWatch) Description() string {
	return "Pull Alarm States from Amazon CloudWatch"
}

// Gather takes in an accumulator and adds the metrics that the Input
// gathers. This is called every "interval".
func (c *CloudWatch) Gather(acc telegraf.Accumulator) error {
	if c.statFilter == nil {
		var err error
		// Set config level filter (won't change throughout life of plugin).
		c.statFilter, err = filter.NewIncludeExcludeFilter(c.TagsInclude, c.TagsExclude)
		if err != nil {
			return err
		}
	}

	if c.client == nil {
		c.initializeCloudWatch()
	}

	wg := sync.WaitGroup{}
	rLock := sync.Mutex{}

	alarms := []*cloudwatch.MetricAlarm{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		result, err := c.DescribeAlarms(c.alarmFilter(c.StateValue))
		if err != nil {
			acc.AddError(err)
			return
		}

		rLock.Lock()
		alarms = result.MetricAlarms
		rLock.Unlock()
	}()
	wg.Wait()

	return c.aggregateAlarms(acc, alarms)

}

func (c *CloudWatch) alarmFilter(stateValue string) *cloudwatch.DescribeAlarmsInput {
	if stateValue == "" {
		stateValue = "ALARM"
	}
	return &cloudwatch.DescribeAlarmsInput{
		StateValue: &stateValue,
	}
}

func (c *CloudWatch) aggregateAlarms(
	acc telegraf.Accumulator,
	metricAlarmResults []*cloudwatch.MetricAlarm,
) error {
	var (
		grouper = metric.NewSeriesGrouper()
	)

	for _, result := range metricAlarmResults {
		tags := map[string]string{}
		tags["region"] = c.Region
		tags["alarmArn"] = *result.AlarmArn
		tags["metricName"] = *result.MetricName
		tags["namespace"] = *result.Namespace
		for i := range result.Dimensions {
			tags[*result.Dimensions[i].Name] = *result.Dimensions[i].Value
		}
		grouper.Add(*result.AlarmName, tags, *result.StateUpdatedTimestamp, "State", *result.StateValue)

	}

	for _, metric := range grouper.Metrics() {
		acc.AddMetric(metric)
	}

	return nil
}

//DescribeAlarms hmm
func (c *CloudWatch) DescribeAlarms(params *cloudwatch.DescribeAlarmsInput) (*cloudwatch.DescribeAlarmsOutput, error) {

	results := []*cloudwatch.MetricAlarm{}

	for {
		resp, err := c.client.DescribeAlarms(params)
		if err != nil {
			fmt.Printf("error %v\n", err)
			return nil, fmt.Errorf("failed to get Alarm data: %v", err)
		}

		results = append(results, resp.MetricAlarms...)
		if resp.NextToken == nil {
			break
		}
		params.NextToken = resp.NextToken
	}
	output := &cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: results,
	}
	return output, nil
}

func (c *CloudWatch) initializeCloudWatch() {
	credentialConfig := &internalaws.CredentialConfig{
		Region:      c.Region,
		AccessKey:   c.AccessKey,
		SecretKey:   c.SecretKey,
		RoleARN:     c.RoleARN,
		Profile:     c.Profile,
		Filename:    c.CredentialPath,
		Token:       c.Token,
		EndpointURL: c.EndpointURL,
	}
	configProvider := credentialConfig.Credentials()

	cfg := &aws.Config{
		HTTPClient: &http.Client{
			// use values from DefaultTransport
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 60 * time.Second,
		},
	}

	loglevel := aws.LogOff
	c.client = cloudwatch.New(configProvider, cfg.WithLogLevel(loglevel))

}

func init() {
	inputs.Add("awsalarms", func() telegraf.Input {
		return &CloudWatch{
			RateLimit: 25,
		}
	})
}
