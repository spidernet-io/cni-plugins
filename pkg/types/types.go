package types

type RPFilter struct {
	// setup host rp_filter
	Enable *bool `json:"set_host,omitempty"`
	// the value of rp_filter, must be 0/1/2
	Value *int32 `json:"value,omitempty"`
}
