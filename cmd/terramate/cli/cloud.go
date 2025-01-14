// Copyright 2023 Terramate GmbH
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rs/zerolog/log"
	"github.com/terramate-io/terramate/cloud"
	"github.com/terramate-io/terramate/cmd/terramate/cli/cliconfig"
	"github.com/terramate-io/terramate/cmd/terramate/cli/out"
	"github.com/terramate-io/terramate/config"
	"github.com/terramate-io/terramate/errors"
	prj "github.com/terramate-io/terramate/project"
)

// ErrOnboardingIncomplete indicates the onboarding process is incomplete.
const ErrOnboardingIncomplete errors.Kind = "cloud commands cannot be used until onboarding is complete"

const defaultCloudTimeout = 5 * time.Second

type cloudConfig struct {
	client *cloud.Client
	output out.O

	credential credential

	run struct {
		runUUID string
		orgUUID string

		meta2id map[string]int
	}
}

type credential interface {
	Name() string
	Load() (bool, error)
	Token() (string, error)
	Refresh() error
	IsExpired() bool
	ExpireAt() time.Time
	Validate(cloudcfg cloudConfig) error
	organizations() cloud.MemberOrganizations
	Info()
}

type keyValue struct {
	key   string
	value string
}

func credentialPrecedence(output out.O, clicfg cliconfig.Config) []credential {
	return []credential{
		newGithubOIDC(output),
		newGoogleCredential(output, clicfg),
	}
}

func (c *cli) checkSyncDeployment() {
	if !c.parsedArgs.Run.CloudSyncDeployment {
		return
	}
	err := c.setupSyncDeployment()
	if err != nil {
		if errors.IsKind(err, ErrOnboardingIncomplete) {
			c.cred().Info()
		}
		fatal(err)
	}

	if orgs := c.cred().organizations(); len(orgs) != 1 {
		fatal(
			errors.E("requires 1 organization associated with the credential but %d found: %s",
				len(orgs),
				orgs),
		)
	}

	c.cloud.run.meta2id = make(map[string]int)

	c.cloud.run.runUUID, err = generateRunID()
	if err != nil {
		fatal(err, "generating run uuid")
	}

	if orgs := c.cloud.credential.organizations(); len(orgs) == 1 {
		c.cloud.run.orgUUID = orgs[0].UUID
	} else {
		fatal(errors.E("expects user associated with a single organization but %d found", len(orgs)))
	}
}

func (c *cli) setupSyncDeployment() error {
	cred, err := c.loadCredential()
	if err != nil {
		return err
	}

	c.cloud = cloudConfig{
		client: &cloud.Client{
			BaseURL:    cloudBaseURL,
			HTTPClient: &http.Client{},
			Credential: cred,
		},
		output:     c.output,
		credential: cred,
	}

	return cred.Validate(c.cloud)
}

func (c *cli) createCloudDeployment(stacks config.List[*config.SortableStack], command []string) {
	logger := log.With().
		Str("organization", c.cloud.run.orgUUID).
		Logger()

	if !c.parsedArgs.Run.CloudSyncDeployment {
		return
	}

	logger.Trace().Msg("Checking if selected stacks have id")

	for _, st := range stacks {
		if st.ID == "" {
			fatal(errors.E("The --cloud-sync-deployment flag requires that selected stacks contain an ID field"))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCloudTimeout)
	defer cancel()

	repoURL, err := c.prj.git.wrapper.URL(c.prj.gitcfg().DefaultRemote)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to retrieve repository URL")
	}

	// TODO(i4k): convert repoURL to Go-style module name (eg.: github.com/org/reponame)

	var payload cloud.DeploymentStacksPayloadRequest
	for _, s := range stacks {
		tags := s.Tags
		if tags == nil {
			tags = []string{}
		}
		payload.Stacks = append(payload.Stacks, cloud.DeploymentStackRequest{
			MetaID:          s.ID,
			MetaName:        s.Name,
			MetaDescription: s.Description,
			MetaTags:        tags,
			Repository:      repoURL,
			Path:            prj.PrjAbsPath(c.rootdir(), c.wd()).String(),
			CommitSHA:       c.prj.git.headCommit,
			Command:         strings.Join(command, " "),
		})
	}
	res, err := c.cloud.client.CreateDeploymentStacks(ctx, c.cloud.run.orgUUID, c.cloud.run.runUUID, payload)
	if err != nil {
		fatal(err)
	}

	if len(res) != len(stacks) {
		err := errors.E("the backend respond with an invalid number of stacks in the deployment: %d instead of %d",
			len(res), len(stacks))

		fatal(err, "unable to continue")
	}

	for _, r := range res {
		logger.Debug().Msgf("deployment created: %+v\n", r)
		if r.StackMetaID == "" {
			fatal(errors.E("backend returned empty meta_id"))
		}
		c.cloud.run.meta2id[r.StackMetaID] = r.StackID
	}
}

func (c *cli) syncCloudDeployment(s *config.Stack, status cloud.Status) {
	logger := log.With().
		Str("organization", c.cloud.run.orgUUID).
		Str("stack", s.RelPath()).
		Stringer("status", status).
		Logger()

	stackID, ok := c.cloud.run.meta2id[s.ID]
	if !ok {
		logger.Error().Msg("unable to update deployment status due to invalid API response")
		return
	}

	payload := cloud.UpdateDeploymentStacks{
		Stacks: []cloud.UpdateDeploymentStack{
			{
				StackID: stackID,
				Status:  status,
			},
		},
	}

	logger.Debug().Msg("updating deployment status")

	ctx, cancel := context.WithTimeout(context.Background(), defaultCloudTimeout)
	defer cancel()
	err := c.cloud.client.UpdateDeploymentStacks(ctx, c.cloud.run.orgUUID, c.cloud.run.runUUID, payload)
	if err != nil {
		logger.Err(err).Str("stack-id", s.ID).Msg("failed to update deployment status for each")
	}
}

func (c *cli) cloudInfo() {
	err := c.setupSyncDeployment()
	if err != nil {
		fatal(err)
	}
	c.cred().Info()
	// verbose info
	c.cloud.output.MsgStdOutV("next token refresh in: %s", time.Until(c.cred().ExpireAt()))
}

func (c *cli) loadCredential() (credential, error) {
	probes := credentialPrecedence(c.output, c.clicfg)
	var cred credential
	var found bool
	for _, probe := range probes {
		var err error
		found, err = probe.Load()
		if err != nil {
			return nil, err
		}
		if found {
			cred = probe
			break
		}
	}
	if !found {
		return nil, errors.E("no credential found")
	}

	return cred, nil
}

func tokenClaims(token string) (jwt.MapClaims, error) {
	jwtParser := &jwt.Parser{}
	tokParsed, _, err := jwtParser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, errors.E(err, "parsing jwt token")
	}

	if claims, ok := tokParsed.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, errors.E("invalid jwt token claims")
}
