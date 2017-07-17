package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var tagName string

func init() {
	flag.StringVar(&tagName, "tagname", "", "name of tag holding the EIP")
}

func main() {
	flag.Parse()
	if tagName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	sess := session.Must(session.NewSession())

	ec2meta := ec2metadata.New(sess)
	instanceID, err := ec2meta.GetMetadata("instance-id")
	if err != nil {
		log.Fatal(err)
	}

	eip, err := getInstanceTag(sess, instanceID, tagName)
	if err != nil {
		log.Fatal(err)
	}

	if err = associateEIPWithInstance(sess, eip, instanceID); err != nil {
		log.Fatal(err)
	}
}

func getInstanceTag(sess *session.Session, instanceID, tagName string) (string, error) {
	svc := ec2.New(sess)

	describeInput := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(instanceID),
				},
			},
		},
	}

	tags, err := svc.DescribeTags(describeInput)
	if err != nil {
		return "", err
	}

	for _, tagDesc := range tags.Tags {
		if *tagDesc.Key == tagName {
			return *tagDesc.Value, nil
		}
	}

	return "", errors.New("tag not found")
}

func associateEIPWithInstance(sess *session.Session, ip, instanceID string) error {
	svc := ec2.New(sess)

	addressInput := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("public-ip"),
				Values: []*string{
					aws.String(ip),
				},
			},
		},
	}

	addressOutput, err := svc.DescribeAddresses(addressInput)
	if err != nil {
		return err
	}

	if len(addressOutput.Addresses) == 0 {
		return fmt.Errorf("couldn't find public ip: %s", ip)
	}

	assocInput := &ec2.AssociateAddressInput{
		AllocationId: addressOutput.Addresses[0].AllocationId,
		InstanceId:   aws.String(instanceID),
	}

	_, err = svc.AssociateAddress(assocInput)
	if err != nil {
		return err
	}

	return nil
}
