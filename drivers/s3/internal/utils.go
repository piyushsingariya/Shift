package driver

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/utils"
)

func newSession(credentials interface{}) (*session.Session, error) {
	if credentials == nil {
		return nil, fmt.Errorf("credentials found nil")
	}
	// assume role session
	if ok, _ := utils.IsOfType(credentials, "account_id"); ok {
		logger.Info("Assume Role credetials found")
		creds := &AssumeRoleAWS{}
		if err := utils.Unmarshal(credentials, creds); err != nil {
			return nil, err
		}

		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(creds.Region),
			Credentials: awscredentials.NewStaticCredentials(creds.AccessKey, creds.SecretAccessKey, ""),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create aws session: %s", err)
		}

		svc := sts.New(sess)

		params := &sts.AssumeRoleInput{
			RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::%s:role/%s", creds.AccountID, creds.RoleName)),
			RoleSessionName: aws.String("manager-assume"),
		}

		assumedRoleOutput, err := svc.AssumeRole(params)
		if err != nil {
			return nil, fmt.Errorf("failed to assume role[%s] for aws-account-id[%s]; %s", creds.RoleName, creds.AccountID, err)
		}

		assumedCreds := assumedRoleOutput.Credentials

		sess, err = session.NewSession(&aws.Config{
			Region: aws.String(creds.Region), // Replace with your desired region
			Credentials: awscredentials.NewStaticCredentials(
				*assumedCreds.AccessKeyId,
				*assumedCreds.SecretAccessKey,
				*assumedCreds.SessionToken,
			),
		})
		if err != nil {
			return nil, err
		}

		return sess, err
	}

	logger.Info("Creating AWS Session")
	creds := &BaseAWS{}
	if err := utils.Unmarshal(credentials, creds); err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(creds.Region),
		Credentials: awscredentials.NewStaticCredentials(creds.AccessKey, creds.SecretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create aws session: %s", err)
	}

	return sess, err
}
