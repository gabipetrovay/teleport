package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/types/externalcloudaudit"
	"github.com/gravitational/teleport/lib/backend"
	"github.com/gravitational/teleport/lib/modules"
	"github.com/gravitational/teleport/lib/services/local"
)

// TODO(tobiaszheller): better name.
type ExternalCloudAuditConfigurator struct {
	mu               sync.Mutex
	config           *externalcloudaudit.ExternalCloudAuditSpec
	awsOIDCGenerator TokenGenerator
	region           string
	bk               backend.Backend
}

func NewExternalCloudAuditConfigurator(ctx context.Context, bk backend.Backend) *ExternalCloudAuditConfigurator {
	if !modules.GetModules().Features().Cloud {
		return nil
	}

	// TODO(tobiaszheller): consider adding some mechanism to disable it
	// via env flag or some other solution.

	config, err := getExternalCloudAuditConfig(ctx, bk)
	if err != nil {
		if errors.Is(err, errExternalAuditNotConfigured) {
			return nil
		}
	}
	return &ExternalCloudAuditConfigurator{
		config: config,
		// TODO(tobiaszheller): load from teleport config.
		region: "us-west-2",
		bk:     bk,
	}
}

func (p *ExternalCloudAuditConfigurator) GetExternalAuditConfig() *externalcloudaudit.ExternalCloudAuditSpec {
	return p.config
}

func (p *ExternalCloudAuditConfigurator) CredentialsV1() *credentials.Credentials {
	// TODO(tobiaszheller): move provider to separate struct.
	return credentials.NewCredentials(p)
}

func (p *ExternalCloudAuditConfigurator) Retrieve() (credentials.Value, error) {
	credsV2, err := p.generateNewCredentials(context.Background())
	if err != nil {
		return credentials.Value{}, trace.Wrap(err)
	}
	return credentials.Value{
		AccessKeyID:     credsV2.AccessKeyID,
		SecretAccessKey: credsV2.SecretAccessKey,
		SessionToken:    credsV2.SessionToken,
		ProviderName:    "aws-oidc",
	}, nil
}

// TODO(tobiaszheller): rework, for now generate always new credentials.
func (c *ExternalCloudAuditConfigurator) IsExpired() bool {
	return false
}

type TokenGenerator interface {
	GenerateAWSOIDCTokenForExternalAudit(context.Context) (string, error)
}

func (p *ExternalCloudAuditConfigurator) SetAWSOIDCGenerator(in TokenGenerator) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.awsOIDCGenerator = in
}

func (p *ExternalCloudAuditConfigurator) GetAWSOIDCGenerator() TokenGenerator {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.awsOIDCGenerator
}

func (p *ExternalCloudAuditConfigurator) isGeneratorReady() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.awsOIDCGenerator != nil
}

func (p *ExternalCloudAuditConfigurator) Start(ctx context.Context) {
	// TODO(tobiaszheller): ExternalAuditCredentialsProvider.start should
	// take care of creating and refresing expiring credentials.
}

func (p *ExternalCloudAuditConfigurator) generateNewCredentials(ctx context.Context) (*aws.Credentials, error) {
	if !p.isGeneratorReady() {
		return nil, trace.Errorf("generator not set yet")
	}
	token, err := p.GetAWSOIDCGenerator().GenerateAWSOIDCTokenForExternalAudit(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	// TODO(tobiaszheller): use grpc service instead of local?
	integrationSvc, err := local.NewIntegrationsService(p.bk)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	integration, err := integrationSvc.GetIntegration(ctx, p.config.IntegrationName)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	awsoidcSpec := integration.GetAWSOIDCIntegrationSpec()
	if awsoidcSpec == nil {
		return nil, trace.BadParameter("missing spec fields for %q (%q) integration", integration.GetName(), integration.GetSubKind())
	}
	roleARN := awsoidcSpec.RoleARN

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.region))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	roleProvider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(cfg),
		roleARN,
		IdentityToken(token),
		func(wiro *stscreds.WebIdentityRoleOptions) {
			wiro.Duration = time.Hour
		},
	)
	creds, err := roleProvider.Retrieve(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &creds, nil
}

// IdentityToken is an implementation of [stscreds.IdentityTokenRetriever] for returning a static token.
type IdentityToken string

// GetIdentityToken returns the token configured.
func (j IdentityToken) GetIdentityToken() ([]byte, error) {
	return []byte(j), nil
}

var errExternalAuditNotConfigured = errors.New("cluster external cloud audit not configured")

func getExternalCloudAuditConfig(ctx context.Context, bk backend.Backend) (*externalcloudaudit.ExternalCloudAuditSpec, error) {
	svc, err := local.NewExternalCloudAuditService(bk)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	externalAudit, err := svc.GetClusterExternalCloudAudit(ctx)
	if err != nil {
		if trace.IsNotFound(err) {
			return nil, errExternalAuditNotConfigured
		}
		return nil, trace.Wrap(err)
	}
	return &externalAudit.Spec, nil
}
