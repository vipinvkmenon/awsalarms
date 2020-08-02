# Amazon CloudWatch Alarms Input

This plugin will pull alarm stats from Amazon CloudWatch

### Motivation

Often time there is a need to be able to pull Alarm status from AWS with the possible
restriction of not being able to integrate with the existing notification
solutions available with AWS. There would sometimes be needs to have poll mechanism
to handle the same

This simple shim can serve as an external telegraf plugin that is capable of polling
in clourwatch alarms at regular intervals.
Also, it serve as a reference application of external telegraf plugins

Derived from the existing implementation of the [cloudwatch input plugin](https://github.com/influxdata/telegraf/blob/effe112473a6bd8991ef8c12e293353c92f1d538/plugins/inputs/cloudwatch/cloudwatch.go)

Just as the `cloudwatch input` plugin, it  uses a credential chain for Authentication with the CloudWatch
API endpoint. In the following order the plugin will attempt to authenticate.
1. Assumed credentials via STS if `role_arn` attribute is specified (source credentials are evaluated from subsequent rules)
2. Explicit credentials from `access_key`, `secret_key`, and `token` attributes
3. Shared profile from `profile` attribute
4. [Environment Variables](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#environment-variables)
5. [Shared Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#shared-credentials-file)
6. [EC2 Instance Profile](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)

NOTE: The Cloudwatch user must have the appropriate permissions to read/Describe Cloudwatch Alarms.

### Configuration

The minimal configuration expects the `region` to be set.
```toml
[[inputs.awsalarms]]
  region = "us-east-1"
  ## Optionals
  ## Amazon Credentials
  ## Credentials are loaded in the following order
  ## 1) Assumed credentials via STS if role_arn is specified
  ## 2) explicit credentials from 'access_key' and 'secret_key'
  ## 3) shared profile from 'profile'
  ## 4) environment variables
  ## 5) shared credentials file
  ## 6) EC2 Instance Profile
  # secret_key = ""
  # token = ""
  # role_arn = ""
  # profile = ""
  # shared_credential_file = ""
  # state_value = "ALARM"
  # tags_include = ["tags-to-include"]
  # tags_exclude = ["tags_to_exclude"]
```
The default state_value is "ALARM"
Tags here refer to the attributes of the Alarm: `Dimensions`, `MetricName`, `Namespace` & `ArnName`
The measurement name being the `AlarmName`

Note: Currently CompositeAlarms not implemented.

Poll interval can be set with the param  `-poll_interval`.</br>
Default Poll Interval is 2 minutes.

### Installation

* Clone the repo

```
git clone git@github.com:vipinvkmenon/awsalarms.git
```
* Build the "awsalarm" binary

```
$ go build -o awsalarm cmd/main.go
```
* You should be able to call this from telegraf now using execd
```
[[inputs.execd]]
  command = ["/path/to/awsalarm", "-poll_interval 1m"]
  signal = "none"
```
This self-contained plugin is based on the documentations of [Execd Go Shim](https://github.com/influxdata/telegraf/blob/effe112473a6bd8991ef8c12e293353c92f1d538/plugins/common/shim/README.md)


