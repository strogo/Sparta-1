+++
author = "Matt Weagle"
date = "2015-11-29T06:50:17"
title = "Custom Resources"
tags = ["sparta"]
type = "doc"
+++

# Introduction

In some circumstances your service may need to provision or access resources that fall outside the standard workflow.  In this case you can use [CloudFormation Lambda-backed CustomResources](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html) to create or access resources during your CloudFormation stack's lifecycle.

Sparta provides unchecked access to the CloudFormation resource lifecycle via the [RequireCustomResource](https://godoc.org/github.com/mweagle/Sparta#LambdaAWSInfo.RequireCustomResource) function.  This function registers a user-supplied [CustomResourceFunction](https://godoc.org/github.com/mweagle/Sparta#CustomResourceFunction) with the larger CloudFormation resource [lifecycle](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-requesttypes.html). 

In this section we'll walk through a sample user-defined custom resource and touch on on way state may be communicated across your application's resources.

## Components

There are two required, and two optional components to user-defined CustomResources:

  1. The user-defined [CustomResourceFunction](https://godoc.org/github.com/mweagle/Sparta#CustomResourceFunction) instance
    - This function defines your resource's logic.  The multiple return value is `map[string]interface{}, error` which signify resource results and operation error, respectively.
  1. The `LambdaAWSInfo` struct which declares a dependency on your custom resource via the [RequireCustomResource](https://godoc.org/github.com/mweagle/Sparta#LambdaAWSInfo.RequireCustomResource) member function.
  1. (Optional) - The template decorator that binds your CustomResource's [data results](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-responses.html) to the owning `LambdaAWSInfo` caller.
  1. (Optional) - A call from your standard Lambda's function body to discover the CustomResource outputs via `sparta.Discover()`. 
  
  
### CustomResourceFunction

This is the core of your custom resource's logic and is executed in a manner similar to standard Sparta functions.  The primary difference is the function signature: 
  
    type CustomResourceFunction func(requestType string
         stackID string
         properties map[string]interface{}
         logger *logrus.Logger) 
    (map[string]interface{}, error)

where

  * `requestType`: The CustomResource [operation type](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-requests.html)
  * `stackID` : The current stack being operated on.  
  * `properties`: User-defined properties provided to [RequireCustomResource](https://godoc.org/github.com/mweagle/Sparta#LambdaAWSInfo.RequireCustomResource) (see below).
  * `logger` : Preconfigured logger instance
  
The multiple return values denote success with non-empty results, or an error. 

As an example, we'll use the following custom resource function:

```
// User defined λ-backed CloudFormation CustomResource
func userDefinedCustomResource(requestType string,
	stackID string,
	properties map[string]interface{},
	logger *logrus.Logger) (map[string]interface{}, error) {

	var results = map[string]interface{}{
		"CustomResourceResult": "Victory!",
	}
	return results, nil
}
```

### RequireCustomResource

The next step is to associate this custom resource function with a previously created Sparta `NewLambda` result via [RequireCustomResource](https://godoc.org/github.com/mweagle/Sparta#LambdaAWSInfo.RequireCustomResource).  This function accepts:

  * `roleNameOrIAMRoleDefinition`: The IAM role name or definition under which the custom resource function should be executed. Equivalent to the same argument in [NewLambda](https://godoc.org/github.com/mweagle/Sparta#NewLambda).
  * `userFunc`: Custom resource function pointer
  * `lambdaOptions`: Lambda execution options. Equivalent to the same argument in [NewLambda](https://godoc.org/github.com/mweagle/Sparta#NewLambda).
  * `resourceProps`: Arbitrary, optional properties that will be provided to the `userFunc` during execution.
  
The multiple return values denote the logical, stable CloudFormation resource ID of the new custom resource, or an error if one occurred.

For example, our custom resource function above can be associated via:


```
// Standard AWS λ function
func helloWorld(event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	configuration, _ := Discover()

	logger.WithFields(logrus.Fields{
		"Discovery": configuration,
	}).Info("Custom resource request")

	fmt.Fprint(w, "Hello World")
}

func ExampleLambdaAWSInfo_RequireCustomResource() {

	lambdaFn := NewLambda(IAMRoleDefinition{},
		helloWorld,
		nil)

	cfResName, _ := lambdaFn.RequireCustomResource(IAMRoleDefinition{},
		userDefinedCustomResource,
		nil,
		nil)

```

These two steps are sufficient to include your custom resource function in the CloudFormation stack lifecycle.  

It's possible to share state from the custom resource to a standard Sparta lambda function by annotating your Sparta lambda function's metadata and then discovering it at execution time.   

### Optional - Template Decorator

The first step is to include a [TemplateDecorator](https://godoc.org/github.com/mweagle/Sparta#TemplateDecorator) that annotates your Sparta lambda functions CloudFormation resource metadata.  This function specifies which user defined output keys (`CustomResourceResult` in this example) you wish to make available during your lambda function's execution.  

```
lambdaFn.Decorator = func(lambdaResourceName string,
  lambdaResource gocf.LambdaFunction,
  resourceMetadata map[string]interface{},
  template *gocf.Template,
  logger *logrus.Logger) error {

  // Pass CustomResource outputs to the λ function
  resourceMetadata["CustomResource"] = gocf.GetAtt(cfResName, "CustomResourceResult")
  return nil
}
```

The `cfResName` value is the CloudFormation resource name returned by `RequireCustomResource`.  The template decorator specifies which of your [CustomResourceFunction](https://godoc.org/github.com/mweagle/Sparta#CustomResourceFunction) outputs should be discoverable during the paren't lambda functions execution time.  

### Optional - Discovery

Discovery is handled by [sparta.Discover()](https://godoc.org/github.com/mweagle/Sparta#Discover) which returns a [DiscoveryInfo](https://godoc.org/github.com/mweagle/Sparta#DiscoveryInfo) pointer containing the linked Custom Resource outputs.  The calling Sparta lambda function can discover it's own [DiscoveryResource](https://godoc.org/github.com/mweagle/Sparta#DiscoveryResource) element via the `ResourceID` field. Once this is found, the calling function can lookup the linked custom resource output via the `Properties` field using the keyname  (`CustomResource`) provided in the previous template decorator.  

In this example, the unmarshalled _DiscoveryInfo_ struct looks like:
  

```json
{
  "Discovery": {
    "ResourceID": "mainhelloWorldLambda837e49c53be175a0f75018a148ab6cd22841cbfb",
    "Region": "us-west-2",
    "StackID": "arn:aws:cloudformation:us-west-2:123412341234:stack/SpartaHelloWorld/70b28170-13f9-11e6-b0c7-50d5ca11b8d2",
    "StackName": "SpartaHelloWorld",
    "Resources": {
      "mainhelloWorldLambda837e49c53be175a0f75018a148ab6cd22841cbfb": {
          "ResourceID": "mainhelloWorldLambda837e49c53be175a0f75018a148ab6cd22841cbfb",
          "Properties": {
            "CustomResource": "Victory!"
          },
          "Tags": {}
      }
    }
  },
  "level": "info",
  "msg": "Custom resource request",
  "time": "2016-05-07T14:13:37Z"
}
```

## Wrapping Up

CloudFormation Custom Resources are a powerful tool that can help pre-existing applications migrate to a Sparta application.  


# Notes
  * Sparta uses [Lambda-backed CustomResource](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html) functions, so they are subject to the same [Lambda limits](http://docs.aws.amazon.com/lambda/latest/dg/limits.html) as application-level Sparta lambda functions.
  * Returning an error from the CustomResourceFunction will result in a _FAILED_ reason being returned in the CloudFormation [response object](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-responses.html).
 