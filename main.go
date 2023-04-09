package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/ipthomas/tukcnst"
	"github.com/ipthomas/tukdbint"
	"github.com/ipthomas/tukdsub"
	"github.com/ipthomas/tukutil"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var initSrvcs = false

func main() {
	lambda.Start(Handle_Request)
}
func Handle_Request(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var err error
	var notify string
	var dbconn tukdbint.TukDBConnection
	if !initSrvcs {
		dbconn = tukdbint.TukDBConnection{DBUser: os.Getenv(tukcnst.ENV_DB_USER), DBPassword: os.Getenv(tukcnst.ENV_DB_PASSWORD), DBHost: os.Getenv(tukcnst.ENV_DB_HOST), DBPort: os.Getenv(tukcnst.ENV_DB_PORT), DBName: os.Getenv(tukcnst.ENV_DB_NAME)}
		if err = tukdbint.NewDBEvent(&dbconn); err != nil {
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
		initSrvcs = true
	}
	log.Printf("Processing API Gateway %s Request Path %s", req.HTTPMethod, req.Path)
	if notify, err = extractEventElement(req.Body); err != nil {
		log.Println(err.Error())
		return queryResponse(http.StatusOK, tukcnst.GO_TEMPLATE_DSUB_ACK, tukcnst.SOAP_XML)
	}
	trans := tukdsub.DSUBEvent{
		PDQ_SERVER_URL:  os.Getenv(tukcnst.PIX_URL),
		PDQ_SERVER_TYPE: tukcnst.PDQ_SERVER_TYPE_IHE_PIXM,
		REG_OID:         os.Getenv(tukcnst.REGION_OID),
		NHS_OID:         tukcnst.NHS_OID_DEFAULT,
		EventMessage:    notify,
		DBConnection:    dbconn,
	}
	tukdsub.New_Transaction(&trans)
	log.Println("Returning DSUB ACK Response")
	return queryResponse(http.StatusOK, tukcnst.GO_TEMPLATE_DSUB_ACK, tukcnst.SOAP_XML)
}

func extractEventElement(msg string) (string, error) {
	if msg == "" {
		return "", errors.New("body is empty")
	}
	notifyElement := tukutil.GetXMLNodeList(msg, tukcnst.DSUB_NOTIFY_ELEMENT)
	if notifyElement == "" {
		return "", errors.New("unable to locate  notify Element")
	}
	log.Println("Extracted Notify Element.")
	log.Println(notifyElement)
	return notifyElement, nil
}
func setAwsResponseHeaders(contentType string) map[string]string {
	awsHeaders := make(map[string]string)
	awsHeaders["Server"] = "TUK_XDW_Consumer_Proxy"
	awsHeaders["Access-Control-Allow-Origin"] = "*"
	awsHeaders["Access-Control-Allow-Headers"] = "accept, Content-Type"
	awsHeaders["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	awsHeaders[tukcnst.CONTENT_TYPE] = contentType
	return awsHeaders
}
func queryResponse(statusCode int, body string, contentType string) (*events.APIGatewayProxyResponse, error) {
	log.Println(body)
	return &events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    setAwsResponseHeaders(contentType),
		Body:       body,
	}, nil
}
