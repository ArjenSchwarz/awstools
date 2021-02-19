package drawio

import (
	_ "embed"
	"encoding/json"
	"log"
)

//go:embed shapes/aws.json
var awsraw []byte
var awsshapes map[string]map[string]string

//AWSShape returns the shape for a desired service
//TODO: Add error handling for unfound shapes
func AWSShape(group string, title string) string {
	shapes := AllAWSShapes()
	return shapes[group][title]
}

//AllAWSShapes returns the full map of shapes
func AllAWSShapes() map[string]map[string]string {
	if awsshapes == nil {
		err := json.Unmarshal(awsraw, &awsshapes)
		if err != nil {
			log.Println(err)
		}
	}
	return awsshapes
}
