package web

const ContentType = "application/problem+json"

type ProblemJSON struct {
	// A URI reference [RFC3986] that identifies the problem type.
	// This specification encourages that, when dereferenced, it provides human-readable documentation for the
	// problem type.
	Type string `json:"type,omitempty"`
	// A short, human-readable summary of the problem type.
	Title string `json:"title"`
	Key   string `json:"key"`
	// The HTTP status code [RFC7231].
	Status int `json:"status"`
	// A human-readable explanation specific to this occurrence of the problem.
	Detail string `json:"detail,omitempty"`
	// A URI reference that identifies the specific occurrence of the problem.
	Instance string `json:"instance,omitempty"`
}

func NewProblemJSON(title string, key string, status int) *ProblemJSON {
	return &ProblemJSON{
		Title:  title,
		Key:    key,
		Status: status,
	}
}
