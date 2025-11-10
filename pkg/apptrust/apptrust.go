package apptrust

import (
	"fmt"

	"github.com/samber/lo"
)

type apptrustError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e apptrustError) String() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

type AppTrustErrorsResponse struct {
	Errors []apptrustError `json:"errors"`
}

func (r AppTrustErrorsResponse) String() string {
	errs := lo.Reduce(r.Errors, func(err string, item apptrustError, _ int) string {
		if err == "" {
			return item.String()
		} else {
			return fmt.Sprintf("%s, %s", err, item.String())
		}
	}, "")
	return errs
}
