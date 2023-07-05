package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Configurable variables
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
		// Create an EC2 instance
		instance, err := ec2.NewInstance(ctx, "my-instance", &ec2.InstanceArgs{
			InstanceType: pulumi.String(instanceType),
			Ami:          pulumi.String(alx.Id), // Replace with the desired AMI ID
			SubnetId:     pulumi.String(my_subnet.Id),
			RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
				VolumeSize: pulumi.Int(ebsVolumeSize),
			},
			AssociatePublicIpAddress: pulumi.Bool(false),
			Tags:                     defaultTags,
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
