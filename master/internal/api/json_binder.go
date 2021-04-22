package api

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MaybeInt allows JSON users to distinguish between absent and null values. It should be used as a
// non-pointer value in structs. After unmarshaling, IsPresent will be true if the corresponding
// key is present, whether or not the value is null; Value will be nil or not, depending on the
// value.
//
// Based on:
// https://www.calhoun.io/how-to-determine-if-a-json-key-has-been-set-to-null-or-not-provided/
type MaybeInt struct {
	IsPresent bool
	Value     *int
}

// UnmarshalJSON unmarshals the given data, which should be an integer or null.
func (i *MaybeInt) UnmarshalJSON(data []byte) error {
	// If this method is called at all, the key must be present.
	i.IsPresent = true

	// Now examine the value; either it's null or we try to unmarshal it as an int.
	if string(data) == "null" {
		i.Value = nil
		return nil
	}

	var temp int
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = &temp
	return nil
}

// BindPatch binds the request body of PATCH requests to the provided interface.
func BindPatch(i interface{}, c echo.Context) error {
	req := c.Request()
	contentType := req.Header.Get(echo.HeaderContentType)

	if req.Method != echo.PATCH || contentType != "application/merge-patch+json" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"can only bind to `application/merge-patch+json` requests")
	}
	if err := json.NewDecoder(req.Body).Decode(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
