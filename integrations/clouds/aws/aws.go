package aws

// MakeAWS creates an AWS cloud struct
func MakeAWS() *AWS {
	return &AWS{
		Version: version,
	}
}
