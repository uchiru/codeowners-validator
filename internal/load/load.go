package load

import (
	"context"

	"go.szostok.io/codeowners-validator/internal/check"
	"go.szostok.io/codeowners-validator/internal/envconfig"
	"go.szostok.io/codeowners-validator/internal/github"

	"github.com/pkg/errors"
)

// For now, it is a good enough solution to init checks. Important thing is to do not require env variables
// and do not create clients which will not be used because of the given checker.
//
// MAYBE in the future the https://github.com/uber-go/dig will be used.
func Checks(ctx context.Context, enabledChecks, experimentalChecks []string) ([]check.Checker, error) {
	var checks []check.Checker

	if isEnabled(enabledChecks, "syntax") {
		checks = append(checks, check.NewValidSyntax())
	}

	if isEnabled(enabledChecks, "duppatterns") {
		checks = append(checks, check.NewDuplicatedPattern())
	}

	if isEnabled(enabledChecks, "files") {
		checks = append(checks, check.NewFileExist())
	}

	if isEnabled(enabledChecks, "owners") {
		var cfg struct {
			OwnerChecker check.ValidOwnerConfig
			Github       github.ClientConfig
		}
		if err := envconfig.Init(&cfg); err != nil {
			return nil, errors.Wrapf(err, "while loading config for %s", "owners")
		}

		ghClient, isApp, err := github.NewClient(ctx, &cfg.Github)
		if err != nil {
			return nil, errors.Wrap(err, "while creating GitHub client")
		}

		owners, err := check.NewValidOwner(cfg.OwnerChecker, ghClient, !isApp)
		if err != nil {
			return nil, errors.Wrap(err, "while enabling 'owners' checker")
		}

		if err := owners.CheckSatisfied(ctx); err != nil {
			return nil, errors.Wrap(err, "while checking if 'owners' checker is satisfied")
		}

		checks = append(checks, owners)
	}

	expChecks, err := loadExperimentalChecks(experimentalChecks)
	if err != nil {
		return nil, errors.Wrap(err, "while loading experimental checks")
	}

	return append(checks, expChecks...), nil
}

func loadExperimentalChecks(experimentalChecks []string) ([]check.Checker, error) {
	var checks []check.Checker

	if contains(experimentalChecks, "notowned") {
		var cfg struct {
			NotOwnedChecker check.NotOwnedFileConfig
		}
		if err := envconfig.Init(&cfg); err != nil {
			return nil, errors.Wrapf(err, "while loading config for %s", "notowned")
		}

		checks = append(checks, check.NewNotOwnedFile(&cfg.NotOwnedChecker))
	}

	if contains(experimentalChecks, "avoid-shadowing") {
		checks = append(checks, check.NewAvoidShadowing())
	}

	return checks, nil
}

func isEnabled(checks []string, name string) bool {
	// if a user does not specify concrete checks then all checks are enabled
	if len(checks) == 0 {
		return true
	}

	if contains(checks, name) {
		return true
	}
	return false
}

func contains(checks []string, name string) bool {
	for _, c := range checks {
		if c == name {
			return true
		}
	}
	return false
}
