package awsalarms

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/assert"
)

type mockGatherCloudWatchClient struct{}

func (m *mockGatherCloudWatchClient) DescribeAlarms(params *cloudwatch.DescribeAlarmsInput) (*cloudwatch.DescribeAlarmsOutput, error) {

	results := []*cloudwatch.MetricAlarm{}
	dimension := &cloudwatch.Dimension{
		Name:  aws.String("vm-instance"),
		Value: aws.String("vm1"),
	}
	dimensions := []*cloudwatch.Dimension{}
	dimensions = append(dimensions, dimension)

	result := &cloudwatch.MetricAlarm{
		AlarmName:             aws.String("alarm"),
		AlarmDescription:      aws.String("alarm"),
		Namespace:             aws.String("AWS/RDS"),
		StateValue:            aws.String("ALARM"),
		AlarmArn:              aws.String("arn:TEST"),
		MetricName:            aws.String("memory"),
		StateUpdatedTimestamp: aws.Time(time.Now()),
		Dimensions:            dimensions,
	}
	results = append(results, result)
	output := &cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: results,
	}
	return output, nil
}
func TestGather(t *testing.T) {

	c := &CloudWatch{
		Region:    "us-east-1",
		AccessKey: "asas",
		SecretKey: "asas",
	}

	var acc testutil.Accumulator
	c.client = &mockGatherCloudWatchClient{}

	assert.NoError(t, acc.GatherError(c.Gather))

	fields := map[string]interface{}{}
	fields["State"] = "ALARM"

	tags := map[string]string{}
	tags["region"] = "us-east-1"
	tags["namespace"] = "AWS/RDS"
	tags["alarmArn"] = "arn:TEST"
	tags["metricName"] = "memory"
	tags["vm-instance"] = "vm1"

	assert.True(t, acc.HasMeasurement("alarm"))
	acc.AssertContainsTaggedFields(t, "alarm", fields, tags)
}
