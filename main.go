package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/joho/godotenv"
)

func main() {
	d := flag.Int("d", 0, "set delaySeconds when sendMessage API is called: defalut is 0")
	w := flag.Int("w", 0, "wait until specified seconds: default is 0")

	flag.Parse()
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
		DelaySeconds: aws.Int64(int64(*d)),
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
		WaitTimeSeconds:     aws.Int64(int64(*w)),
	}

	var stdout io.Writer = os.Stdout

	ch := receiveMessageWrapper(svc, input)
	for {
		select {
		case result := <-ch:
			if len(result.Messages) == 0 {
				continue
			}
			json.NewEncoder(stdout).Encode(result.Messages[0])

			_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      input.QueueUrl,
				ReceiptHandle: result.Messages[0].ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("[err] %v", err)
			}
			return
		case <-time.After(10 * time.Second):
			fmt.Println("time out")
			return
		}
	}
}

/*
* keep receiving message from a queue until it returns valid message
 */
func receiveMessageWrapper(svc *sqs.SQS, input *sqs.ReceiveMessageInput) <-chan *sqs.ReceiveMessageOutput {
	ch := make(chan *sqs.ReceiveMessageOutput, 1)
	go func() {
		for {
			msg := receiveMessage(svc, input)
			if len(msg.Messages) == 0 {
				fmt.Println("message is empty")
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			ch <- msg
		}
	}()
	return ch
}

func receiveMessage(svc *sqs.SQS, input *sqs.ReceiveMessageInput) *sqs.ReceiveMessageOutput {

	msgResult, err := svc.ReceiveMessage(input)
	if err != nil {
		fmt.Printf("[err] %v", err)
		return &sqs.ReceiveMessageOutput{}
	}

	return msgResult
}
