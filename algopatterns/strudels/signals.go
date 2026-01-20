package strudels

import "context"

// returns the CC Signal for a strudel
func (r *Repository) GetStrudelCCSignal(ctx context.Context, strudelID string) (*CCSignal, error) {
	var ccSignal *CCSignal

	err := r.db.QueryRow(ctx, queryGetParentCCSignal, strudelID).Scan(&ccSignal)
	if err != nil {
		return nil, err
	}

	return ccSignal, nil
}

// returns the parent strudel ID if this is a fork
func (r *Repository) GetStrudelForkedFrom(ctx context.Context, strudelID string) (*string, error) {
	var forkedFrom *string

	err := r.db.QueryRow(ctx, queryGetStrudelForkedFrom, strudelID).Scan(&forkedFrom)
	if err != nil {
		return nil, err
	}

	return forkedFrom, nil
}

// checks if the user owns any strudel with the exact code
// used for paste detection validation - loading own strudel shouldn't trigger paste lock
func (r *Repository) UserOwnsStrudelWithCode(ctx context.Context, userID, code string) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx, queryUserOwnsStrudelWithCode, userID, code).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// checks if any public strudel has the exact code AND allows AI
// used for paste detection validation - forking a public strudel that allows AI shouldn't trigger paste lock
// BUT forking a public strudel with no-ai CC signal SHOULD trigger paste lock to protect creator's wishes
func (r *Repository) PublicStrudelExistsWithCodeAllowsAI(ctx context.Context, code string) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx, queryPublicStrudelExistsWithCodeAllowsAI, code).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// checks if any public strudel has the exact code with no-ai CC signal
func (r *Repository) PublicStrudelExistsWithCodeNoAI(ctx context.Context, code string) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx, queryPublicStrudelExistsWithCodeNoAI, code).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
