package sparta

import (
	"context"
)

const LambdaExecuteARN = "LambdaExecutor"
const s3BucketSourceArn = "arn:aws:s3:::sampleBucket"
const snsTopicSourceArn = "arn:aws:sns:us-west-2:000000000000:someTopic"
const dynamoDBTableArn = "arn:aws:dynamodb:us-west-2:000000000000:table/sampleTable"

func mockLambda1(ctx context.Context) (string, error) {
	return "mockLambda1!", nil
}

func mockLambda2(ctx context.Context) (string, error) {
	return "mockLambda2!", nil
}

func mockLambda3(ctx context.Context) (string, error) {
	return "mockLambda3!", nil
}

func testLambdaData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 1
	lambdaFn := HandleAWSLambda(LambdaName(mockLambda1),
		mockLambda1,
		LambdaExecuteARN)
	lambdaFn.Permissions = append(lambdaFn.Permissions, S3Permission{
		BasePermission: BasePermission{
			SourceArn: s3BucketSourceArn,
		},
		// Event Filters are defined at
		// http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html
		Events: []string{"s3:ObjectCreated:*", "s3:ObjectRemoved:*"},
	})

	lambdaFn.Permissions = append(lambdaFn.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopicSourceArn,
		},
	})

	lambdaFn.EventSourceMappings = append(lambdaFn.EventSourceMappings, &EventSourceMapping{
		StartingPosition: "TRIM_HORIZON",
		EventSourceArn:   dynamoDBTableArn,
		BatchSize:        10,
	})

	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 2
	lambdaFunctions = append(lambdaFunctions, HandleAWSLambda(LambdaName(mockLambda2),
		mockLambda2,
		LambdaExecuteARN))
	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 3
	// https://github.com/mweagle/Sparta/pull/1
	lambdaFn3 := HandleAWSLambda(LambdaName(mockLambda3),
		mockLambda3,
		LambdaExecuteARN)
	lambdaFn3.Permissions = append(lambdaFn3.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopicSourceArn,
		},
	})
	lambdaFunctions = append(lambdaFunctions, lambdaFn3)

	return lambdaFunctions
}
