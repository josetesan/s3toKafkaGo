package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	router := httprouter.New()
	router.PUT("/", sendFileToS3)
	err := http.ListenAndServe("0.0.0.0:3000", router)
	if err != nil {
		log.Fatal(err)
	}

}

func sendFileToS3(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	filename, err := FileUpload(r)
	//here we call the function we made to get the image and save it
	if err != nil {
		http.Error(w, "Invalid Data", http.StatusBadRequest)
		return
		//checking whether any error occurred retrieving image
	}

	id, err := s3Upload("s3bucket", filename)
	if err != nil {
		http.Error(w, "Invalid Data on uploading s3: "+err.Error(), http.StatusBadRequest)
		return
	}

	err2 := sendMessageToKafka(id, "TestTopic")
	if err2 != nil {
		http.Error(w, "Invalid Data on sending message to kafka: "+err2.Error(), http.StatusBadRequest)
		return
	}
	print("File uploaded !\n")
}

func FileUpload(r *http.Request) (string, error) {
	//this function returns the filename(to save in database) of the saved file or an error if it occurs
	r.ParseMultipartForm(32 << 20)
	//ParseMultipartForm parses a request body as multipart/form-data
	file, handler, err := r.FormFile("file") //retrieve the file from form data
	//replace file with the key your sent your image with
	if err != nil {
		return "Error", err
	}
	defer file.Close() //close the file when we finish
	//this is path which  we want to store the file
	f, err := os.OpenFile("/tmp/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "Error", err
	}
	defer f.Close()
	io.Copy(f, file)
	//here we save our file to our path
	fmt.Printf("File Received %v\n", f.Name())
	return f.Name(), nil
}

func s3Upload(bucketName string, filename string) (string, error) {

	// Create Transport
	tr := &http.Transport{
		ResponseHeaderTimeout: 1 * time.Second,
	}

	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession(&aws.Config{
		HTTPClient: &http.Client{Transport: tr},
		// retrieves them from
		//* Access Key ID: AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY
		//* Secret Access Key: AWS_SECRET_ACCESS_KEY or AWS_SECRET_KEY
		Credentials: credentials.NewEnvCredentials(),
		Region:      aws.String("eu-west-1"),
		Endpoint:    aws.String("http://127.0.0.1:9000"),
		LogLevel:    aws.LogLevel(aws.LogDebugWithEventStreamBody),
		S3ForcePathStyle: aws.Bool(true),
	}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		return "Error reading file", err
	}
	defer f.Close()

	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("mykey"),
		Body:   f,

	}

	// Perform upload with options different than the those in the Uploader.
	result, err := uploader.Upload(upParams, func(u *s3manager.Uploader) {
		u.PartSize = 50 * 1024 * 1024 // 50MB part size
		u.LeavePartsOnError = true    // Don't delete the parts if the upload fails.
		u.Concurrency = 4
	})

	if err != nil {
		return "Can't upload", err
	}

	print("File Uploaded to s3, id=" + result.UploadID)

	return result.UploadID, err

}

func sendMessageToKafka(id string, topic string) error {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "kafka-dev-0.europe.intranet:9092"})
	if err != nil {
		return err
	}
	fmt.Printf("Created Producer %v\n", p)
	defer p.Close()

	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message at %v\n", ev.Timestamp)
				}
			}
		}
	}()

	// Produce messages to topic (asynchronously)
	p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(id),
		Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
	}, nil)

	// Wait for message deliveries before shutting down
	p.Flush(15 * 1000)

	return nil
}
