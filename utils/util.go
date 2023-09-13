package utils

import (
	"github.com/google/uuid"
	//"github.com/chromedp/cdproto/har"
)

func GenerateUUID() string {
	return uuid.NewString()
}

/*
 * Generate .har file
 */
