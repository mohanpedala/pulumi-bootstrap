package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Configurable variables
		scriptDir := "scripts"
		instanceType := "t2.micro" // Instance type
		ebsVolumeSize := 20        // EBS volume size in GB
		defaultTags := pulumi.StringMap{
			"Name": pulumi.String("pulumibootstrap"),
		}

		// VPC lookup
		my_vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
			Tags: map[string]string{
				"Name": "default",
			},
		}, nil)
		if err != nil {
			return err
		}
		// Subnet ID lookup
		my_subnet, err := ec2.LookupSubnet(ctx, &ec2.LookupSubnetArgs{
			VpcId: &my_vpc.Id,
			Filters: []ec2.GetSubnetFilter{
				ec2.GetSubnetFilter{
					Name: "availability-zone",
					Values: []string{
						"us-east-1a",
					},
				},
			},
		}, nil)
		if err != nil {
			return err
		}
		// Search for amazon linux 2023 ami by applying filters
		alx, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
			MostRecent: pulumi.BoolRef(true),
			Filters: []ec2.GetAmiFilter{
				ec2.GetAmiFilter{
					Name: "name",
					Values: []string{
						"al2023-ami-2023.1.20230629.0-*",
					},
				},
				ec2.GetAmiFilter{
					Name: "virtualization-type",
					Values: []string{
						"hvm",
					},
				},
			},
			Owners: []string{
				"137112412989",
			},
		}, nil)
		if err != nil {
			return err
		}

		// Create an IAM role for the EC2 instance and access an S3 bucket to list objects
		ec2Role, err := iam.NewRole(ctx, "ec2-instance-role", &iam.RoleArgs{
			Name: pulumi.String("ec2-instance-role"),
			Tags: defaultTags,
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {
							"Service": "ec2.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
		})
		if err != nil {
			return err
		}

		// Create an inline policy to allow listing objects in an S3 bucket
		_, err = iam.NewRolePolicy(ctx, "ec2-instance-policy", &iam.RolePolicyArgs{
			Role: ec2Role.Name,
			Policy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": [
							"s3:ListBucket"
						],
						"Resource": [
							"arn:aws:s3:::my-bucket"
						]
					}
				]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach policy to the role
		_, err = iam.NewRolePolicyAttachment(ctx, "ec2-instance-policy-attachment", &iam.RolePolicyAttachmentArgs{
			Role:      ec2Role.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSSMFullAccess"),
		})
		if err != nil {
			return err
		}

		// Create an instance profile for the EC2 instance
		ec2roleInstanceProfile, err := iam.NewInstanceProfile(ctx, "ec2roleInstanceProfile", &iam.InstanceProfileArgs{
			Name: pulumi.String("ec2roleInstanceProfile"),
			Role: ec2Role.Name,
		})
		if err != nil {
			return err
		}

		userDataFiles, err := filepath.Glob(filepath.Join(scriptDir, "*.py"))
		if err != nil {
			return err
		}

		var userData string
		for _, file := range userDataFiles {
			content, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}
			userData += string(content)
		}

		// Create an EC2 instance
		instance, err := ec2.NewInstance(ctx, "my-instance", &ec2.InstanceArgs{
			InstanceType: pulumi.String(instanceType),
			Ami:          pulumi.String(alx.Id), // Replace with the desired AMI ID
			SubnetId:     pulumi.String(my_subnet.Id),
			RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
				VolumeSize: pulumi.Int(ebsVolumeSize),
			},
			IamInstanceProfile:       ec2roleInstanceProfile.Name,
			AssociatePublicIpAddress: pulumi.Bool(false),
			Tags:                     defaultTags,
			UserData:                 pulumi.String(userData),
		})
		if err != nil {
			return err
		}

		// Create a new EBS volume
		addVolume, err := ebs.NewVolume(ctx, "my-volume", &ebs.VolumeArgs{
			Size:             pulumi.Int(ebsVolumeSize),
			AvailabilityZone: pulumi.String(my_subnet.AvailabilityZone),
			Tags:             defaultTags,
		})
		if err != nil {
			return err
		}

		// Mount EBS volume to EC2 instance

		_, err = ec2.NewVolumeAttachment(ctx, "my-ebs-volume-attachment", &ec2.VolumeAttachmentArgs{
			DeviceName:                  pulumi.String("/dev/sdh"),
			InstanceId:                  instance.ID(),
			VolumeId:                    addVolume.ID(),
			SkipDestroy:                 pulumi.Bool(false),
			StopInstanceBeforeDetaching: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Create an S3 bucket
		_, err = s3.NewBucket(ctx, "my-bucket", &s3.BucketArgs{
			Acl:  pulumi.String("private"),
			Tags: defaultTags,
		})

		if err != nil {
			return err
		}
		// Upload a Python program to the EC2 instance
		publicIP := instance.PublicIp
		privateIP := instance.PrivateIp

		ctx.Export("publicIP", publicIP)
		ctx.Export("privateIP", privateIP)

		return nil
	})
}
