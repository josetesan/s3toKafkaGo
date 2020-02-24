package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/julienschmidt/httprouter"
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
	//a function to change a profile image

	imageName, err := FileUpload(r)
	//here we call the function we made to get the image and save it
	if err != nil {
		http.Error(w, "Invalid Data", http.StatusBadRequest)
		return
		//checking whether any error occurred retrieving image
	}

	print(imageName)
}

func FileUpload(r *http.Request) (string, error) {
	//this function returns the filename(to save in database) of the saved file or an error if it occurs
	r.ParseMultipartForm(32 << 20)
	//ParseMultipartForm parses a request body as multipart/form-data
	file, handler, err := r.FormFile("file") //retrieve the file from form data
	//replace file with the key your sent your image with
	if err != nil {
		return "", err
	}
	defer file.Close() //close the file when we finish
	//this is path which  we want to store the file
	f, err := os.OpenFile("/tmp/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	io.Copy(f, file)
	//here we save our file to our path
	return handler.Filename, nil
}

func s3Upload(bucketName string, keyName string, file os.File) {

	// Create Transport
	tr := &http.Transport{
		ResponseHeaderTimeout: 1 * time.Second,
	}

	// The session the S3 Uploader will use

	sess := session.Must(session.NewSession(&aws.Config{
		HTTPClient: &http.Client{Transport: tr},
	}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	var r io.Reader
	r = file

	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &keyName,
		Body:   file,
	}

	// Perform upload with options different than the those in the Uploader.
	result, err := uploader.Upload(upParams, func(u *s3manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 10MB part size
		u.LeavePartsOnError = true    // Don't delete the parts if the upload fails.
	})
}
