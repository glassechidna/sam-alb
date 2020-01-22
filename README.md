# AWS SAM Application Load Balancer macro

The [AWS Serverless Application Model][sam] (SAM) defines a simplified framework
on top of CloudFormation for building serverless apps. It has great support for
AWS API Gateway, but [so far no support][gh-issue] for Application Load Balancers.

This is unfortunate as ALBs are very handy! Depending on your definition of 
"serverless", they may or may not qualify: there is a minimum cost of about $20
per month, even if you use nothing. But there are several benefits relative to
API Gateway REST APIs and HTTP APIs:

* Cheaper at high volume. In this [breakdown], the author estimates their ALB
  bill to be $166 instead of $4,163 per month for an API Gateway REST API serving
  450 requests per second. The (currently beta) HTTP API would about $1,000/month.

* Seamlessly transition from a serverful environment of EC2 or container targets
  behind an ALB to a Lambda behind that same ALB incrementally. ALBs let you
  shift traffic percentage-wise, or direct traffic based on host names, headers,
  paths, query strings or source IP addresses.
  
* Enterprise-friendly. API Gateway has some limitations in its ability to be locked
  down in granular ways, so large enterprises often block it outright. Those same
  enterprises usually have no problem with internal ALBs - so you are no longer
  locked out of a serverless future.
  
* Better (in my opinion) support for authentication through OpenID Connect and
  Cognito. While these are supported by API Gateway, they don't handle redirection
  of unauthenticated users.
  
* Sticky weighted traffic. An ALB can make A/B testing easier by splitting traffic
  between multiple Lambdas - and ensuring that subsequent requests from the same
  client are forwarded to the same Lambda. 
  
## Installation

This operates as a CloudFormation [macro][macro]. That means you must "install"
it into your account (on a regional basis) before you can use it in your stacks.

**TODO**: Here I will write about how you can install it from the 
[Serverless App Repository][sar]. Need to work that out first.

## Usage

Usage is pretty simple. You can see a complete example in [`demo.yml`](/demo.yml).
There are two parts to it. First, you need to add a reference to the macro in
the list of transforms in your template. 

Replace this:

    Transform: "AWS::Serverless-2016-10-31"
    
With this:

    Transform: ["SamAlb", "AWS::Serverless-2016-10-31"]

Note that the order matters - `SamAlb` needs to come first. Next, you can now
use a new event type. Here's what that looks like:

```yaml
  Function:
    Type: AWS::Serverless::Function
    Properties:
      Handler: index.handler
      Runtime: python3.7
      Events:
        Alb:
          Type: ALB
          Properties:
            ListenerArn: !Ref ListenerArn
            Priority: 50
            Conditions:
              Host: [example.com]
```

The value for `ListenerArn` should be a string that is of the form 
`arn:aws:elasticloadbalancing:$REGION:$ACCOUNT_ID:listener/app/$ALBNAME/$RANDOM_HEX/$RANDOM_HEX`.

The complete set of properties for the `ALB` event type are:

| Property name | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| `ListenerArn` | Type: String. **Required**. Should be the ARN of an [`AWS::ElasticLoadBalancingV2::Listener`][listener] |
| `Priority`    | Type: Integer. **Required**. This needs to be a value between 1 and 50,000 inclusive. The priority must be unique (i.e. two events can't both be priority 100) and is used to determine rule order evaluation. Rules are evaluated in ascending order. |
| `Oidc`        | Type: [`AuthenticateOidcConfig`][cfn-oidc]. Optional. The properties of this type are the same as the CloudFormation type. One difference is that `AuthorizationEndpoint`, `TokenEndpoint` and `UserInfoEndpoint` are optional. If not provided, they are determined using the [well-known endpoint][well-known] of the `Issuer`. |
| `Conditions`  | Type: Array of `Condition`. **Required**. There must be at least one condition specified, i.e. an empty array is insufficient. |

The `Condition` type has the following properties:

| Property name | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| `Host`        | Type: Array of strings. Optional.                            |
| `Path`        | Type: Array of strings. Optional.                            |
| `Method`      | Type: Array of strings. Optional.                            |
| `Ip`          | Type: Array of strings. Optional.                            |
| `Header`      | Type: Key-value map. Values are arrays of strings. Optional. |

Every property is optional, but **at least one** must be specified. Evaluation is 
similar to IAM policies: objects are ANDed and arrays are OR. The following example 
will match `example.com/hello` and `amazon.com/hello`:

```
Conditions:
  - Host: [example.com, amazon.com]
    Path: [/hello]
```

## Roadmap

The following is planned:

* Support for _creating_ ALBs. Right now, the ALB must be created elsewhere - in
  the same template is fine. The plan is to support `VpcConfig` with the same structure
  as the `AWS::Serverless::Function` type.
  
* Route 53 record set creation. Currently records must be created in Route 53
  elsewhere.
  
* Cognito authentication support. 

[sam]: https://github.com/awslabs/serverless-application-model
[gh-issue]: https://github.com/awslabs/serverless-application-model/issues/721
[breakdown]: https://serverless-training.com/articles/save-money-by-replacing-api-gateway-with-application-load-balancer/
[macro]: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-macros.html
[sar]: https://aws.amazon.com/serverless/serverlessrepo/
[listener]: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html
[cfn-oidc]: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-authenticateoidcconfig.html
[well-known]: https://ldapwiki.com/wiki/Openid-configuration
