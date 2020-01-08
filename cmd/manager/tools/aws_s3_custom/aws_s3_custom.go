package aws_s3_custom

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

// PolicyDocument is our definition of our policies to be uploaded to IAM.
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

// StatementEntry will dictate what this policy will allow or not allow.
type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

// This function creates a folder with the sprovided filename in the given S3 bucketname
func CreateFolderIfNotExist(accessKeyID, secretAccessKey, filename, bucketName, region string) (success bool) {
	success = false
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})

	// Create S3 service client
	svc := s3.New(sess)

	_, err = svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})

	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == s3.ErrCodeNoSuchKey {
		_, err := svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(filename),
		})

		if err != nil {
			fmt.Println("Create Folder error", err)
			return
		}

		success = true
		return
	}
	if err != nil {
		fmt.Println("GetFolder Error", err)
		return
	}
	return
}

// Creates a User if not already present with the provided username, returns user credentials
func CreateUserIfNotExist(accessKeyID, secretAccessKey, userName, region string) (awsAccessKey string, awsSecretAccessKey string, success bool) {
	success = false
	awsAccessKey = ""
	awsSecretAccessKey = ""

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})

	// Create a IAM service client.
	svc := iam.New(sess)

	_, err = svc.GetUser(&iam.GetUserInput{
		UserName: &userName,
	})

	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == iam.ErrCodeNoSuchEntityException {
		_, err := svc.CreateUser(&iam.CreateUserInput{
			UserName: &userName,
		})

		if err != nil {
			fmt.Println("CreateUser Error", err)
			return
		}

		accessKeyResult, accessKeyErr := svc.CreateAccessKey(&iam.CreateAccessKeyInput{
			UserName: aws.String(userName),
		})

		if accessKeyErr != nil {
			fmt.Println("Error", accessKeyErr)
			return
		}

		success = true
		awsAccessKey = *accessKeyResult.AccessKey.AccessKeyId
		awsSecretAccessKey = *accessKeyResult.AccessKey.SecretAccessKey
		return
	} else {
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		result, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
			MaxItems: aws.Int64(5),
			UserName: aws.String(userName),
		})
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		for _, b := range result.AccessKeyMetadata {
			awsAccessKey = *b.AccessKeyId
		}
	}
	return
}

// CreateKeyIfNotExist is used to create key for the username if key not exists, returns new keys
func CreateKeyIfNotExist(accessKeyID, secretAccessKey, userName, region string) (awsAccessKey string, awsSecretAccessKey string, success bool) {
	success = false
	awsAccessKey = ""
	awsSecretAccessKey = ""

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})

	// Create a IAM service client.
	svc := iam.New(sess)
	result, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
		MaxItems: aws.Int64(5),
		UserName: aws.String(userName),
	})
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	for _, b := range result.AccessKeyMetadata {
		// fmt.Printf("* %s access key deleted\n",
		// 	aws.StringValue(b.AccessKeyId))
		svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			AccessKeyId: b.AccessKeyId,
			UserName:    &userName,
		})
	}
	accessKeyResult, accessKeyErr := svc.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	})

	if accessKeyErr != nil {
		fmt.Println("Error", accessKeyErr)
		return
	}

	success = true
	awsAccessKey = *accessKeyResult.AccessKey.AccessKeyId
	awsSecretAccessKey = *accessKeyResult.AccessKey.SecretAccessKey
	return
}

// Returns AWS Account Number for the provided user credentials
func getUserAccountNumber(accessKeyID, secretAccessKey, region string) (accountNumber string) {
	accountNumber = ""
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	svc := sts.New(sess)
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	accountNumber = *result.Account
	return
}

// Attaches provided policy to the provided username
func attachPolicyToUser(accessKeyID, secretAccessKey, region, policyArn, userName string) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	svc := iam.New(sess)

	_, err = svc.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: &policyArn,
		UserName:  &userName,
	})

	if err != nil {
		fmt.Println("Unable to attach role policy to user", err)
		return
	}
}

// Create a IAM Policy if not exist and attaches the policy to the user, returns status
func CreatePolicyIfNotExist(accessKeyID, secretAccessKey, filename, bucket, region, userName string) (success bool) {
	success = false
	var policyName = filename[:len(filename)-1] + "_s3_policy"
	var arnString = "arn:aws:s3:::" + bucket + "/" + filename[:len(filename)-1]
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})

	// Create a IAM service client.
	svc := iam.New(sess)

	// Check if the policy exists
	// userPolicyArn := "arn:aws:iam::aws:policy/" + policyName
	userPolicyArn := "arn:aws:iam::" + getUserAccountNumber(accessKeyID, secretAccessKey, region) + ":policy/" + policyName
	result2, err := svc.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: &userPolicyArn,
	})

	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == iam.ErrCodeNoSuchEntityException {
		// Builds our policy document for IAM.
		policy := PolicyDocument{
			Version: "2012-10-17",
			Statement: []StatementEntry{
				StatementEntry{
					Effect: "Allow",
					// Allows for DeleteItem, GetItem, PutItem, Scan, and UpdateItem
					Action: []string{
						"s3:*",
					},
					Resource: arnString,
				},
			},
		}
		b, err := json.Marshal(&policy)
		if err != nil {
			fmt.Println("Error marshaling policy", err)
			return
		}
		result, err := svc.CreatePolicy(&iam.CreatePolicyInput{
			PolicyDocument: aws.String(string(b)),
			PolicyName:     aws.String(policyName),
		})
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		success = true
		// fmt.Println("Policy created and attached to user successfully")
		attachPolicyToUser(accessKeyID, secretAccessKey, region, *result.Policy.Arn, userName)
		return
	}
	if err != nil {
		fmt.Println("Unable to attach role policy to user", err)
		return
	}
	attachPolicyToUser(accessKeyID, secretAccessKey, region, *result2.Policy.Arn, userName)
	// fmt.Println("Policy attached to user successfully")
	success = true
	return
}
