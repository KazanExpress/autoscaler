package yandexcloud

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"

	"github.com/drone/autoscaler"
	"github.com/drone/autoscaler/logger"
)

func (p *provider) Create(ctx context.Context, opts autoscaler.InstanceCreateOpts) (*autoscaler.Instance, error) {
	name := strings.ToLower(opts.Name)

	// select random zone from the list
	zone := p.zone[rand.Intn(len(p.zone))]

	sourceImageID, err := p.getSourceImage(ctx)
	if err != nil {
		return nil, fmt.Errorf("get source image: %w", err)
	}

	log := logger.FromContext(ctx).
		WithField("zone", zone).
		WithField("image", sourceImageID).
		WithField("name", opts.Name)

	op, err := p.service.WrapOperation(p.createInstance(ctx, sourceImageID, p.folderID, zone, name, p.subnetID))
	if err != nil {
		return nil, fmt.Errorf("make wrap operation: %w", err)
	}

	meta, err := op.Metadata()
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	log.Debugf("Creating instance %s\n", meta.(*compute.CreateInstanceMetadata).InstanceId)
	err = op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait create operation: %w", err)
	}

	resp, err := op.Response()
	if err != nil {
		return nil, fmt.Errorf("get opearation response: %w", err)
	}

	ycInstance := resp.(*compute.Instance)

	address := ycInstance.NetworkInterfaces[0].PrimaryV4Address.Address

	instance := &autoscaler.Instance{
		Provider: autoscaler.ProviderYandexCloud,
		ID:       ycInstance.Id,
		Name:     opts.Name,
		Image:    sourceImageID,
		Region:   zone,
		Address:  address,
	}

	return instance, nil
}

func (p *provider) createInstance(
	ctx context.Context,
	imageID, folderID, zone, name, subnetID string,
) (*operation.Operation, error) {

	networkConfig := &compute.PrimaryAddressSpec{}

	if !p.privateIP {
		networkConfig = &compute.PrimaryAddressSpec{
			OneToOneNatSpec: &compute.OneToOneNatSpec{
				IpVersion: compute.IpVersion_IPV4,
			},
		}
	}

	metadata := map[string]string{}
	if p.sshUserPublicKeyPair != "" {
		metadata["ssh-keys"] = p.sshUserPublicKeyPair
	}
	if p.dockerComposeMetadata != "" {
		metadata["docker-compose"] = p.dockerComposeMetadata
	}
	metadata["user-data"] = `#cloud-config
write_files:
- path: /etc/systemd/system/docker.service.d/override.conf
  permissions: '0644'
  content: |
  [Service]
  ExecStart=
  ExecStart=/usr/bin/dockerd
- path: /etc/default/docker
  permissions: '0644'
  content: |
  DOCKER_OPTS=""
- path: /etc/docker/daemon.json
  permissions: '0644'
  content: |
  {
    "hosts": [ "0.0.0.0:2376", "unix:///var/run/docker.sock" ],
    "tls": true,
    "tlsverify": true,
    "tlscacert": "/etc/docker/ca.pem",
    "tlscert": "/etc/docker/server-cert.pem",
    "tlskey": "/etc/docker/server-key.pem"
  }
- path: /etc/docker/ca.pem
  permissions: '0644'
  encoding: b64
  content: {{ .CACert | base64 }}
- path: /etc/docker/server-cert.pem
  permissions: '0644'
  encoding: b64
  content: {{ .TLSCert | base64 }}
- path: /etc/docker/server-key.pem
  permissions: '0644'
  encoding: b64
  content: {{ .TLSKey | base64 }}

runcmd:
  - sudo systemctl daemon-reload
  - sudo systemctl restart docker.service
`

	request := &compute.CreateInstanceRequest{
		FolderId:   folderID,
		Name:       name,
		ZoneId:     zone,
		Metadata:   metadata,
		PlatformId: p.platformID,
		ResourcesSpec: &compute.ResourcesSpec{
			Cores:        p.resourceCores,
			Memory:       p.resourceMemory,
			CoreFraction: p.resourceCoreFraction,
		},
		BootDiskSpec: &compute.AttachedDiskSpec{
			AutoDelete: true,
			Disk: &compute.AttachedDiskSpec_DiskSpec_{
				DiskSpec: &compute.AttachedDiskSpec_DiskSpec{
					TypeId: p.diskType,
					Size:   p.diskSize,
					Source: &compute.AttachedDiskSpec_DiskSpec_ImageId{
						ImageId: imageID,
					},
				},
			},
		},
		NetworkInterfaceSpecs: []*compute.NetworkInterfaceSpec{
			{
				SubnetId:             subnetID,
				PrimaryV4AddressSpec: networkConfig,
				SecurityGroupIds:     p.securityGroupIDs,
			},
		},
		SchedulingPolicy: &compute.SchedulingPolicy{Preemptible: p.preemptible},
	}

	op, err := p.service.Compute().Instance().Create(ctx, request)
	return op, err
}

func (p *provider) getSourceImage(ctx context.Context) (string, error) {
	image, err := p.service.Compute().Image().GetLatestByFamily(ctx, &compute.GetImageLatestByFamilyRequest{
		FolderId: p.imageFolderID,
		Family:   p.imageFamily,
	})
	if err != nil {
		return "", fmt.Errorf("get image id: %w", err)
	}

	return image.Id, nil
}
