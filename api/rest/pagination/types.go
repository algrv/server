package pagination

// Params holds pagination parameters from request
type Params struct {
	Limit  int
	Offset int
}

// Meta holds pagination metadata for response
type Meta struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// NewMeta creates pagination metadata from params and total count
func NewMeta(params Params, total int) Meta {
	return Meta{
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: params.Offset+params.Limit < total,
	}
}

// DefaultParams returns pagination params with defaults applied
// defaultLimit: default items per page, maxLimit: maximum allowed limit
func DefaultParams(limit, offset, defaultLimit, maxLimit int) Params {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return Params{
		Limit:  limit,
		Offset: offset,
	}
}
