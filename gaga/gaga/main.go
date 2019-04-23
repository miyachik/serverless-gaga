package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/image/draw"
)

// Handler kicked from lambda executor
func Handler(ctx context.Context, s3Event events.S3Event) {
	service := s3.New(session.New(), aws.NewConfig().WithRegion("ap-northeast-1"))
	for _, record := range s3Event.Records {
		if strings.HasSuffix(record.S3.Object.Key, "-thumbnail") {
			return
		}
		s3Object, buff := getImageFromS3(service, record.S3.Bucket.Name, record.S3.Object.Key)

		output := resizeImage(buff)

		result := putImageToS3(service, record.S3.Bucket.Name, record.S3.Object.Key, s3Object.ContentType, output)
		fmt.Println(result)
	}
	return
}

func main() {
	lambda.Start(Handler)
}

// TODO: err返す
func getImageFromS3(service *s3.S3, bucket string, key string) (*s3.GetObjectOutput, *bytes.Buffer) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := service.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, nil
	}
	defer result.Body.Close()
	wb, brb := new(bytes.Buffer), new(bytes.Buffer)
	brb.ReadFrom(result.Body)
	fmt.Fprint(wb, brb.String())

	fmt.Println(result.Body)

	return result, wb
}

// TODO: err返す
func putImageToS3(service *s3.S3, bucket string, key string, contentType *string, buff *bytes.Buffer) *s3.PutObjectOutput {
	input := &s3.PutObjectInput{
		Body:                 aws.ReadSeekCloser(bytes.NewReader(buff.Bytes())),
		Bucket:               aws.String(bucket),
		Key:                  aws.String(fmt.Sprintf("%s-thumbnail", key)),
		ContentType:          contentType,
		ACL:                  aws.String(s3.BucketCannedACLPublicRead),
		ServerSideEncryption: aws.String("AES256"),
	}

	result, err := service.PutObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}
	return result
}

// TODO: err返す
func resizeImage(buff *bytes.Buffer) *bytes.Buffer {
	imgSrc, format, err := image.Decode(buff)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	rctSrc := imgSrc.Bounds()
	fmt.Println("Width:", rctSrc.Dx())
	fmt.Println("Height:", rctSrc.Dy())

	//scale down by 1/2
	imgDst := image.NewRGBA(image.Rect(0, 0, rctSrc.Dx()/2, rctSrc.Dy()/2))
	draw.CatmullRom.Scale(imgDst, imgDst.Bounds(), imgSrc, rctSrc, draw.Over, nil)
	resizedBuffer := new(bytes.Buffer)
	switch format {
	case "jpeg":
		if err := jpeg.Encode(resizedBuffer, imgDst, &jpeg.Options{Quality: 100}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
	case "gif":
		if err := gif.Encode(resizedBuffer, imgDst, nil); err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
	case "png":
		if err := png.Encode(resizedBuffer, imgDst); err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
	default:
		fmt.Fprintln(os.Stderr, "format error")
	}
	return resizedBuffer
}
