package types

type RPFilter struct {
	// setup host rp_filter
	Enable *bool `json:"enable,omitempty"`
	// the value of rp_filter, must be 0/1/2
	Value *int32 `json:"value,omitempty"`
}
