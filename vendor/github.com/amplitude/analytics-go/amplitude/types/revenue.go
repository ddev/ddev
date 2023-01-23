package types

type Revenue struct {
	Price       float64
	Quantity    int
	ProductID   string
	RevenueType string
	Receipt     string
	ReceiptSig  string
	Properties  map[string]interface{}
	Revenue     float64
}

func (r Revenue) Validate() []string {
	var validateErrors []string
	if r.Revenue == 0 && r.Price == 0 {
		validateErrors = append(validateErrors, "Either Revenue or Price should be set")
	}

	return validateErrors
}
