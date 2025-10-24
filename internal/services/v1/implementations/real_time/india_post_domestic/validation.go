package india_post_domestic

import (
    "fmt"
    "regexp"
)

var pincodeRe = regexp.MustCompile(`^\d{6}$`)

func validatePincode(pin string) error {
    if !pincodeRe.MatchString(pin) {
        return fmt.Errorf("pincode must be 6 digits")
    }
    return nil
}


