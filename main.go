package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/joho/godotenv"
)

func main() {
	sess := session.Must(
		session.NewSession(
			&aws.Config{
				Region: aws.String("ap-northeast-1"),
			},
		),
	)
	svc := sqs.New(sess)

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("cannot load .env :%v", err)
	}
	queueURL := os.Getenv("QUEUE_URL")

	_, err = svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(5),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Title": {
				DataType:    aws.String("String"),
				StringValue: aws.String("The Whistler"),
			},
			"Author": {
				DataType:    aws.String("String"),
				StringValue: aws.String("John Grisham"),
			},
			"WeeksOn": {
				DataType:    aws.String("Number"),
				StringValue: aws.String("6"),
			},
		},
		MessageBody: aws.String("Information about current NY Times fiction bestseller for week of 12/11/2016."),
		QueueUrl:    &queueURL,
	})
	if err != nil {
		fmt.Printf("[err] %v", err)
		return
	}

	input := &sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(5),
	}
	c := make(chan error)
	go receiveMessage(svc, input, c)
	err = <-c
	if err != nil {
		fmt.Printf("[err] %v", err)
		return
	}
}

func receiveMessage(svc *sqs.SQS, input *sqs.ReceiveMessageInput, c chan<- error) {

	msgResult, err := svc.ReceiveMessage(input)
	if err != nil {
		c <- err
		return
	}

	if len(msgResult.Messages) == 0 {
		c <- errors.New("the specified queue does not return any messages")
		return
	}

	var stdout io.Writer = os.Stdout
	json.NewEncoder(stdout).Encode(msgResult)

	_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      input.QueueUrl,
		ReceiptHandle: msgResult.Messages[0].ReceiptHandle,
	})
	if err != nil {
		c <- err
	}
}
