package images

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Storage is an interface that allows vorteil to be easily configured to manage multiple image storage options.
type storage interface {
	Put(string, string) error
	Get(string) (io.ReadCloser, error)
	Delete(string) error
	List() ([]string, error)
}

// StorageConfiguration contains all necessary information to set up a valid storage option for vorteil.
type storageConfiguration struct {
	Type      string          `yaml:"mode"`
	LocalPath string          `yaml:"local_path"`
	S3        s3Configuration `yaml:"amazon_s3"`
}

// S3Configuration provides additional nested information that will be used by storage if the storage type is an Amazon S3 service.
type s3Configuration struct {
	Region string `yaml:"region"`
}

// InitStorage takes a valid storage configuration file and creates the appropriate object implementing storage.
func initStorage(config *storageConfiguration) (storage, error) {

	switch config.Type {
	case "local":
		return newLocalStorage(config.LocalPath)
	case "amazon s3":
		return newS3Storage(config.S3.Region)
	default:
		return nil, errors.New("invalid/no storage type in config file")

	}

}

// localStorage is an implementation of storage that relies on image files being accessible on local hardware.
type localStorage struct {
	path string
}

// newLocalStorage creates a localStorage object.
func newLocalStorage(path string) (*localStorage, error) {

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	err := os.MkdirAll(path, 0777)
	if err != nil {
		return nil, err
	}

	return &localStorage{path: path}, nil

}

// put implements storage and places a file in the storage location.
func (s *localStorage) Put(name, tempfile string) error {
	err := os.Rename(tempfile, s.path+name)
	return err
}

// get implements storage and returns the file in the storage location.
func (s *localStorage) Get(name string) (io.ReadCloser, error) {
	return os.Open(s.path + name)
}

// delete implements storage and removes the file from the storage location.
func (s *localStorage) Delete(name string) error {
	return os.Remove(s.path + name)
}

// list implements storage and returns a list of files stored in storage.
func (s *localStorage) List() ([]string, error) {

	infos, err := ioutil.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, info := range infos {
		files = append(files, info.Name())
	}

	return files, nil

}

// amazonS3 is an implementation of storage that relies on access to an AWS account with S3. Assumes
// credentials are in environment variables or in a hidden credentials file.
type amazonS3 struct {
	S3 *s3.S3
}

// newS3Storage creates and returns an amazonS3 storage object.
func newS3Storage(region string) (*amazonS3, error) {
	s := session.New(&aws.Config{Region: aws.String(region)})
	a := new(amazonS3)
	a.S3 = s3.New(s)

	// check if vorteil bucket exists
	params := &s3.HeadBucketInput{
		Bucket: aws.String("vorteil"),
	}

	_, err := a.S3.HeadBucket(params)

	if err != nil {
		if err.Error()[:18] == "BucketRegionError:" {
			return nil, err
		}

		input := &s3.CreateBucketInput{
			Bucket: aws.String("vorteil"),
		}
		_, err := a.S3.CreateBucket(input)
		if err != nil {
			return nil, err
		}
	}

	return a, nil

}

// put implements storage and uploads the given file to amazon's s3 servers.
func (s *amazonS3) Put(name, tempfile string) error {

	file, err := os.Open(tempfile)
	if err != nil {
		return err
	}
	defer os.Remove(tempfile)
	defer file.Close()

	params := &s3.PutObjectInput{
		Bucket: aws.String("vorteil"),
		Key:    aws.String(name),
		Body:   file,
	}

	_, err = s.S3.PutObject(params)
	if err != nil {
		return err
	}

	return nil

}

// get implements storage and retrieves the given file from amazon's s3 servers.
func (s *amazonS3) Get(name string) (io.ReadCloser, error) {

	params := &s3.GetObjectInput{
		Bucket: aws.String("vorteil"),
		Key:    aws.String(name),
	}

	resp, err := s.S3.GetObject(params)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil

}

// delete implements storage and removes the given file from amazon's s3 servers.
func (s *amazonS3) Delete(name string) error {

	params := &s3.DeleteObjectInput{
		Bucket: aws.String("vorteil"),
		Key:    aws.String(name),
	}

	_, err := s.S3.DeleteObject(params)
	return err

}

// list implements storage and lists all files stored in the vorteil bucket at amazon's s3 servers.
func (s *amazonS3) List() ([]string, error) {

	params := &s3.ListObjectsInput{
		Bucket: aws.String("vorteil"),
	}

	resp, err := s.S3.ListObjects(params)
	if err != nil {
		return nil, err
	}

	var str []string
	for _, element := range resp.Contents {
		str = append(str, *element.Key)
	}

	return str, nil

}
