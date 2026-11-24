package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cognito"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() { pulumi.Run(run) }
func run(ctx *pulumi.Context) error {
	c := config.New(ctx, "")
	project := c.Require("project")
	env := c.Require("environment")
	name := project + "-" + env
	if len(project) > 35 || len(name) > 50 {
		return fmt.Errorf("project must be at most 35 characters and project-environment at most 50 characters")
	}
	tags := pulumi.StringMap{"Name": pulumi.String(name), "Project": pulumi.String(project), "Environment": pulumi.String(env), "Component": pulumi.String("designer"), "Owner": pulumi.String("lambdawalker/site.card-designer"), "ManagedBy": pulumi.String("Pulumi")}
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return err
	}
	identity, err := aws.GetCallerIdentity(ctx, nil)
	if err != nil {
		return err
	}
	domain := c.Get("domain")
	cert := c.Get("certificateArn")
	zone := c.Get("zoneId")
	if domain != "" && cert == "" {
		return fmt.Errorf("domain requires an ISSUED ACM certificate ARN in us-east-1; validate it before preview")
	}
	web, err := s3.NewBucketV2(ctx, "web", &s3.BucketV2Args{Tags: tags})
	if err != nil {
		return err
	}
	data, err := s3.NewBucketV2(ctx, "data", &s3.BucketV2Args{Tags: tags}, pulumi.Protect(true))
	if err != nil {
		return err
	}
	for key, b := range map[string]*s3.BucketV2{"web": web, "data": data} {
		if _, err = s3.NewBucketPublicAccessBlock(ctx, key+"-private", &s3.BucketPublicAccessBlockArgs{Bucket: b.ID(), BlockPublicAcls: pulumi.Bool(true), BlockPublicPolicy: pulumi.Bool(true), IgnorePublicAcls: pulumi.Bool(true), RestrictPublicBuckets: pulumi.Bool(true)}); err != nil {
			return err
		}
		if _, err = s3.NewBucketServerSideEncryptionConfigurationV2(ctx, key+"-encryption", &s3.BucketServerSideEncryptionConfigurationV2Args{Bucket: b.ID(), Rules: s3.BucketServerSideEncryptionConfigurationV2RuleArray{&s3.BucketServerSideEncryptionConfigurationV2RuleArgs{ApplyServerSideEncryptionByDefault: &s3.BucketServerSideEncryptionConfigurationV2RuleApplyServerSideEncryptionByDefaultArgs{SseAlgorithm: pulumi.String("AES256")}}}}); err != nil {
			return err
		}
		if _, err = s3.NewBucketVersioningV2(ctx, key+"-versions", &s3.BucketVersioningV2Args{Bucket: b.ID(), VersioningConfiguration: &s3.BucketVersioningV2VersioningConfigurationArgs{Status: pulumi.String("Enabled")}}); err != nil {
			return err
		}
	}
	table, err := dynamodb.NewTable(ctx, "tokens", &dynamodb.TableArgs{BillingMode: pulumi.String("PAY_PER_REQUEST"), HashKey: pulumi.String("token_hash"), Attributes: dynamodb.TableAttributeArray{&dynamodb.TableAttributeArgs{Name: pulumi.String("token_hash"), Type: pulumi.String("S")}, &dynamodb.TableAttributeArgs{Name: pulumi.String("owner"), Type: pulumi.String("S")}}, GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{&dynamodb.TableGlobalSecondaryIndexArgs{Name: pulumi.String("owner-index"), HashKey: pulumi.String("owner"), ProjectionType: pulumi.String("ALL")}}, Ttl: &dynamodb.TableTtlArgs{AttributeName: pulumi.String("expires_at"), Enabled: pulumi.Bool(true)}, PointInTimeRecovery: &dynamodb.TablePointInTimeRecoveryArgs{Enabled: pulumi.Bool(true)}, Tags: tags}, pulumi.Protect(true))
	if err != nil {
		return err
	}
	pool, err := cognito.NewUserPool(ctx, "users", &cognito.UserPoolArgs{AutoVerifiedAttributes: pulumi.StringArray{pulumi.String("email")}, UsernameAttributes: pulumi.StringArray{pulumi.String("email")}, DeletionProtection: pulumi.String("ACTIVE"), Tags: tags}, pulumi.Protect(true))
	if err != nil {
		return err
	}
	hosted, err := cognito.NewUserPoolDomain(ctx, "login", &cognito.UserPoolDomainArgs{Domain: pulumi.String(name + "-" + identity.AccountId), UserPoolId: pool.ID()})
	if err != nil {
		return err
	}
	api, err := apigatewayv2.NewApi(ctx, "api", &apigatewayv2.ApiArgs{ProtocolType: pulumi.String("HTTP"), Tags: tags})
	if err != nil {
		return err
	}
	oac, err := cloudfront.NewOriginAccessControl(ctx, "web-access", &cloudfront.OriginAccessControlArgs{Name: pulumi.String(name), OriginAccessControlOriginType: pulumi.String("s3"), SigningBehavior: pulumi.String("always"), SigningProtocol: pulumi.String("sigv4")})
	if err != nil {
		return err
	}
	viewer := &cloudfront.DistributionViewerCertificateArgs{CloudfrontDefaultCertificate: pulumi.Bool(true)}
	aliases := pulumi.StringArray{}
	if domain != "" {
		aliases = append(aliases, pulumi.String(domain))
		viewer = &cloudfront.DistributionViewerCertificateArgs{AcmCertificateArn: pulumi.String(cert), SslSupportMethod: pulumi.String("sni-only"), MinimumProtocolVersion: pulumi.String("TLSv1.2_2021")}
	}
	workerHeaders, err := cloudfront.NewResponseHeadersPolicy(ctx, "worker-sandbox", &cloudfront.ResponseHeadersPolicyArgs{Name: pulumi.String(name + "-worker"), SecurityHeadersConfig: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigArgs{ContentSecurityPolicy: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigContentSecurityPolicyArgs{ContentSecurityPolicy: pulumi.String("default-src 'none'; script-src 'self' 'unsafe-eval'; connect-src 'none'; worker-src 'none'"), Override: pulumi.Bool(true)}}})
	if err != nil {
		return err
	}
	dist, err := cloudfront.NewDistribution(ctx, "site", &cloudfront.DistributionArgs{Enabled: pulumi.Bool(true), DefaultRootObject: pulumi.String("index.html"), Aliases: aliases, Origins: cloudfront.DistributionOriginArray{&cloudfront.DistributionOriginArgs{OriginId: pulumi.String("web"), DomainName: web.BucketRegionalDomainName, OriginAccessControlId: oac.ID()}, &cloudfront.DistributionOriginArgs{OriginId: pulumi.String("api"), DomainName: api.ApiEndpoint.ApplyT(func(s string) string { return strings.TrimPrefix(s, "https://") }).(pulumi.StringOutput), CustomOriginConfig: &cloudfront.DistributionOriginCustomOriginConfigArgs{HttpPort: pulumi.Int(80), HttpsPort: pulumi.Int(443), OriginProtocolPolicy: pulumi.String("https-only"), OriginSslProtocols: pulumi.StringArray{pulumi.String("TLSv1.2")}}}}, DefaultCacheBehavior: &cloudfront.DistributionDefaultCacheBehaviorArgs{TargetOriginId: pulumi.String("web"), ViewerProtocolPolicy: pulumi.String("redirect-to-https"), AllowedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")}, CachedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")}, CachePolicyId: pulumi.String("4135ea2d-6df8-44a3-9df3-4b5a84be39ad"), Compress: pulumi.Bool(true)}, OrderedCacheBehaviors: cloudfront.DistributionOrderedCacheBehaviorArray{&cloudfront.DistributionOrderedCacheBehaviorArgs{PathPattern: pulumi.String("/assets/template-worker-*"), TargetOriginId: pulumi.String("web"), ViewerProtocolPolicy: pulumi.String("https-only"), AllowedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")}, CachedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")}, CachePolicyId: pulumi.String("4135ea2d-6df8-44a3-9df3-4b5a84be39ad"), ResponseHeadersPolicyId: workerHeaders.ID()}, &cloudfront.DistributionOrderedCacheBehaviorArgs{PathPattern: pulumi.String("/api/*"), TargetOriginId: pulumi.String("api"), ViewerProtocolPolicy: pulumi.String("https-only"), AllowedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD"), pulumi.String("OPTIONS"), pulumi.String("POST"), pulumi.String("PUT"), pulumi.String("PATCH"), pulumi.String("DELETE")}, CachedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")}, CachePolicyId: pulumi.String("4135ea2d-6df8-44a3-9df3-4b5a84be39ad"), OriginRequestPolicyId: pulumi.String("b689b0a8-53d0-40ab-baf2-68738e2966ac")}}, Restrictions: &cloudfront.DistributionRestrictionsArgs{GeoRestriction: &cloudfront.DistributionRestrictionsGeoRestrictionArgs{RestrictionType: pulumi.String("none")}}, ViewerCertificate: viewer, Tags: tags})
	if err != nil {
		return err
	}
	site := dist.DomainName.ApplyT(func(s string) string {
		if domain != "" {
			s = domain
		}
		return "https://" + s
	}).(pulumi.StringOutput)
	client, err := cognito.NewUserPoolClient(ctx, "browser", &cognito.UserPoolClientArgs{UserPoolId: pool.ID(), GenerateSecret: pulumi.Bool(false), AllowedOauthFlowsUserPoolClient: pulumi.Bool(true), AllowedOauthFlows: pulumi.StringArray{pulumi.String("code")}, AllowedOauthScopes: pulumi.StringArray{pulumi.String("openid"), pulumi.String("email"), pulumi.String("profile")}, SupportedIdentityProviders: pulumi.StringArray{pulumi.String("COGNITO")}, CallbackUrls: pulumi.StringArray{pulumi.Sprintf("%s/", site)}, LogoutUrls: pulumi.StringArray{pulumi.Sprintf("%s/", site)}, PreventUserExistenceErrors: pulumi.String("ENABLED")})
	if err != nil {
		return err
	}
	role, err := iam.NewRole(ctx, "api-role", &iam.RoleArgs{AssumeRolePolicy: pulumi.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}`), Tags: tags})
	if err != nil {
		return err
	}
	logs, err := cloudwatch.NewLogGroup(ctx, "api-logs", &cloudwatch.LogGroupArgs{Name: pulumi.String("/aws/lambda/" + name), RetentionInDays: pulumi.Int(30), Tags: tags})
	if err != nil {
		return err
	}
	policy := pulumi.All(data.Arn, table.Arn, logs.Arn).ApplyT(func(v []interface{}) (string, error) {
		doc := map[string]interface{}{"Version": "2012-10-17", "Statement": []interface{}{map[string]interface{}{"Effect": "Allow", "Action": []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, "Resource": []string{v[0].(string) + "/users/*", v[0].(string) + "/cache/*"}}, map[string]interface{}{"Effect": "Allow", "Action": []string{"s3:ListBucket"}, "Resource": v[0], "Condition": map[string]interface{}{"StringLike": map[string]interface{}{"s3:prefix": []string{"users/*", "cache/*"}}}}, map[string]interface{}{"Effect": "Allow", "Action": []string{"dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"}, "Resource": []string{v[1].(string), v[1].(string) + "/index/owner-index"}}, map[string]interface{}{"Effect": "Allow", "Action": []string{"logs:CreateLogStream", "logs:PutLogEvents"}, "Resource": v[2].(string) + ":*"}}}
		b, e := json.Marshal(doc)
		return string(b), e
	}).(pulumi.StringOutput)
	rp, err := iam.NewRolePolicy(ctx, "api-permissions", &iam.RolePolicyArgs{Role: role.ID(), Policy: policy})
	if err != nil {
		return err
	}
	fn, err := lambda.NewFunction(ctx, "handler", &lambda.FunctionArgs{Name: pulumi.String(name), Role: role.Arn, Runtime: pulumi.String("provided.al2023"), Handler: pulumi.String("bootstrap"), Architectures: pulumi.StringArray{pulumi.String("arm64")}, Code: pulumi.NewFileArchive("../backend/bootstrap.zip"), MemorySize: pulumi.Int(1024), Timeout: pulumi.Int(30), Environment: &lambda.FunctionEnvironmentArgs{Variables: pulumi.StringMap{"DATA_BUCKET": data.Bucket, "TOKEN_TABLE": table.Name, "COGNITO_USER_POOL_ID": pool.ID(), "COGNITO_CLIENT_ID": client.ID(), "COGNITO_DOMAIN": pulumi.Sprintf("https://%s.auth.%s.amazoncognito.com", hosted.Domain, region.Name), "REDIRECT_URI": pulumi.Sprintf("%s/", site), "LOGOUT_URI": pulumi.Sprintf("%s/", site)}}, Tags: tags}, pulumi.DependsOn([]pulumi.Resource{rp, logs}))
	if err != nil {
		return err
	}
	integration, err := apigatewayv2.NewIntegration(ctx, "lambda", &apigatewayv2.IntegrationArgs{ApiId: api.ID(), IntegrationType: pulumi.String("AWS_PROXY"), IntegrationUri: fn.InvokeArn, PayloadFormatVersion: pulumi.String("2.0")})
	if err != nil {
		return err
	}
	if _, err = apigatewayv2.NewRoute(ctx, "api-routes", &apigatewayv2.RouteArgs{ApiId: api.ID(), RouteKey: pulumi.String("ANY /api/{proxy+}"), Target: pulumi.Sprintf("integrations/%s", integration.ID())}); err != nil {
		return err
	}
	if _, err = apigatewayv2.NewStage(ctx, "live", &apigatewayv2.StageArgs{ApiId: api.ID(), Name: pulumi.String("$default"), AutoDeploy: pulumi.Bool(true), Tags: tags}); err != nil {
		return err
	}
	if _, err = lambda.NewPermission(ctx, "gateway", &lambda.PermissionArgs{Action: pulumi.String("lambda:InvokeFunction"), Function: fn.Name, Principal: pulumi.String("apigateway.amazonaws.com"), SourceArn: pulumi.Sprintf("%s/*/*", api.ExecutionArn)}); err != nil {
		return err
	}
	wp := pulumi.All(web.Arn, dist.Arn).ApplyT(func(v []interface{}) (string, error) {
		d := map[string]interface{}{"Version": "2012-10-17", "Statement": []interface{}{map[string]interface{}{"Effect": "Allow", "Principal": map[string]string{"Service": "cloudfront.amazonaws.com"}, "Action": "s3:GetObject", "Resource": v[0].(string) + "/*", "Condition": map[string]interface{}{"StringEquals": map[string]interface{}{"AWS:SourceArn": v[1]}}}, map[string]interface{}{"Effect": "Deny", "Principal": "*", "Action": "s3:*", "Resource": []string{v[0].(string), v[0].(string) + "/*"}, "Condition": map[string]interface{}{"Bool": map[string]string{"aws:SecureTransport": "false"}}}}}
		b, e := json.Marshal(d)
		return string(b), e
	}).(pulumi.StringOutput)
	if _, err = s3.NewBucketPolicy(ctx, "web-policy", &s3.BucketPolicyArgs{Bucket: web.ID(), Policy: wp}); err != nil {
		return err
	}
	if domain != "" && zone != "" {
		if _, err = route53.NewRecord(ctx, "site-dns", &route53.RecordArgs{ZoneId: pulumi.String(zone), Name: pulumi.String(domain), Type: pulumi.String("A"), Aliases: route53.RecordAliasArray{&route53.RecordAliasArgs{Name: dist.DomainName, ZoneId: dist.HostedZoneId, EvaluateTargetHealth: pulumi.Bool(false)}}}); err != nil {
			return err
		}
	}
	err = filepath.Walk("../web/dist", func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if info.IsDir() {
			return nil
		}
		key, _ := filepath.Rel("../web/dist", path)
		key = filepath.ToSlash(key)
		ct := "application/octet-stream"
		switch filepath.Ext(key) {
		case ".html":
			ct = "text/html"
		case ".js":
			ct = "application/javascript"
		case ".css":
			ct = "text/css"
		case ".json":
			ct = "application/json"
		case ".svg":
			ct = "image/svg+xml"
		case ".png":
			ct = "image/png"
		}
		_, e = s3.NewBucketObjectv2(ctx, "asset-"+key, &s3.BucketObjectv2Args{Bucket: web.ID(), Key: pulumi.String(key), Source: pulumi.NewFileAsset(path), ContentType: pulumi.String(ct), CacheControl: pulumi.String("no-cache, max-age=0, must-revalidate")})
		return e
	})
	if err != nil {
		return err
	}
	if _, err = s3.NewBucketPolicy(ctx, "data-tls", &s3.BucketPolicyArgs{Bucket: data.ID(), Policy: data.Arn.ApplyT(func(a string) string {
		return fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Principal":"*","Action":"s3:*","Resource":["%s","%s/*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}`, a, a)
	}).(pulumi.StringOutput)}); err != nil {
		return err
	}
	ctx.Export("siteUrl", site)
	ctx.Export("distributionId", dist.ID())
	ctx.Export("dataBucket", data.Bucket)
	ctx.Export("tokenTable", table.Name)
	ctx.Export("cognitoClientId", client.ID())
	return nil
}
