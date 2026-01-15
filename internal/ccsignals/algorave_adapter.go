package ccsignals

import (
	"context"

	"codeberg.org/algorave/server/algorave/strudels"
)

// implements ContentValidator using Algorave's strudels repository
type StrudelValidator struct {
	repo *strudels.Repository
}

// creates a ContentValidator backed by the strudels repository
func NewStrudelValidator(repo *strudels.Repository) *StrudelValidator {
	return &StrudelValidator{repo: repo}
}

// checks if the user owns a strudel with this exact code
func (v *StrudelValidator) ValidateOwnership(ctx context.Context, userID, code string) (*ContentMatch, error) {
	owns, err := v.repo.UserOwnsStrudelWithCode(ctx, userID, code)
	if err != nil {
		return nil, err
	}

	if !owns {
		return &ContentMatch{Found: false}, nil
	}

	return &ContentMatch{
		Found:   true,
		OwnerID: userID,
	}, nil
}

// checks if code matches any public strudel
func (v *StrudelValidator) ValidatePublicContent(ctx context.Context, code string) (*ContentMatch, error) {
	allowsAI, err := v.repo.PublicStrudelExistsWithCodeAllowsAI(ctx, code)
	if err != nil {
		return nil, err
	}

	if allowsAI {
		return &ContentMatch{
			Found:    true,
			IsPublic: true,
			CCSignal: SignalCredit,
		}, nil
	}

	existsNoAI, err := v.repo.PublicStrudelExistsWithCodeNoAI(ctx, code)
	if err != nil {
		return &ContentMatch{Found: false}, nil
	}

	if existsNoAI {
		return &ContentMatch{
			Found:    true,
			IsPublic: true,
			CCSignal: SignalNoAI,
		}, nil
	}

	return &ContentMatch{Found: false}, nil
}
