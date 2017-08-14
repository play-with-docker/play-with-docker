package provisioner

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
	"github.com/play-with-docker/play-with-docker/storage"
)

var asgService *autoscaling.AutoScaling
var ec2Service *ec2.EC2

func init() {
	// Create a session to share configuration, and load external configuration.
	sess := session.Must(session.NewSession())
	//
	// // Create the service's client with the session.
	asgService = autoscaling.New(sess)
	ec2Service = ec2.New(sess)
}

type windows struct {
	factory docker.FactoryApi
	storage storage.StorageApi
}

type instanceInfo struct {
	publicIP  string
	privateIP string
	id        string
}

func NewWindowsASG(f docker.FactoryApi, st storage.StorageApi) *windows {
	return &windows{factory: f, storage: st}
}

func (d *windows) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {

	conf.ImageName = config.GetSSHImage()

	winfo, err := d.getWindowsInstanceInfo(session.Id)

	if err != nil {
		return nil, err
	}

	if conf.Hostname == "" {
		var nodeName string
		for i := 1; ; i++ {
			nodeName = fmt.Sprintf("node%d", i)
			exists := checkHostnameExists(session, nodeName)
			if !exists {
				break
			}
		}
		conf.Hostname = nodeName
	}

	containerName := fmt.Sprintf("%s_%s", session.Id[:8], conf.Hostname)
	opts := docker.CreateContainerOpts{
		Image:           conf.ImageName,
		WindowsEndpoint: winfo.privateIP,
		SessionId:       session.Id,
		PwdIpAddress:    session.PwdIpAddress,
		ContainerName:   containerName,
		Hostname:        conf.Hostname,
		ServerCert:      conf.ServerCert,
		ServerKey:       conf.ServerKey,
		CACert:          conf.CACert,
		Privileged:      false,
		HostFQDN:        conf.Host,
	}

	dockerClient, err := d.factory.GetForSession(session.Id)
	if err != nil {
		d.releaseInstance(session.Id, winfo.id)
		return nil, err
	}
	_, err = dockerClient.CreateContainer(opts)
	if err != nil {
		d.releaseInstance(session.Id, winfo.id)
		return nil, err
	}

	instance := &types.Instance{}
	instance.Image = opts.Image
	instance.IP = winfo.privateIP
	instance.SessionId = session.Id
	instance.WindowsId = winfo.id
	instance.Cert = conf.Cert
	instance.Key = conf.Key
	instance.Type = conf.Type
	instance.ServerCert = conf.ServerCert
	instance.ServerKey = conf.ServerKey
	instance.CACert = conf.CACert
	instance.Session = session
	instance.ProxyHost = router.EncodeHost(session.Id, instance.IP, router.HostOpts{})
	instance.SessionHost = session.Host
	// For now this condition holds through. In the future we might need a more complex logic.
	instance.IsDockerHost = opts.Privileged

	if cli, err := d.factory.GetForInstance(instance); err != nil {
		d.InstanceDelete(session, instance)
		return nil, err
	} else {
		info, err := cli.GetDaemonInfo()
		if err != nil {
			d.InstanceDelete(session, instance)
			return nil, err
		}
		instance.Hostname = info.Name
		instance.Name = fmt.Sprintf("%s_%s", session.Id[:8], info.Name)
		if err = dockerClient.ContainerRename(containerName, instance.Name); err != nil {
			d.InstanceDelete(session, instance)
			return nil, err
		}
	}

	return instance, nil

}

func (d *windows) InstanceDelete(session *types.Session, instance *types.Instance) error {
	dockerClient, err := d.factory.GetForSession(session.Id)
	if err != nil {
		return err
	}
	err = dockerClient.DeleteContainer(instance.Name)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		return err
	}

	// TODO trigger deletion in AWS
	return d.releaseInstance(session.Id, instance.WindowsId)
}

func (d *windows) releaseInstance(sessionId, instanceId string) error {
	return d.storage.InstanceDeleteWindows(sessionId, instanceId)
}

func (d *windows) InstanceResizeTerminal(instance *types.Instance, rows, cols uint) error {
	dockerClient, err := d.factory.GetForSession(instance.SessionId)
	if err != nil {
		return err
	}
	return dockerClient.ContainerResize(instance.Name, rows, cols)
}

func (d *windows) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	dockerClient, err := d.factory.GetForSession(instance.SessionId)
	if err != nil {
		return nil, err
	}
	return dockerClient.CreateAttachConnection(instance.Name)
}

func (d *windows) InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error {
	return fmt.Errorf("Not implemented")
}

func (d *windows) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	return fmt.Errorf("Not implemented")
}

func (d *windows) getWindowsInstanceInfo(sessionId string) (*instanceInfo, error) {

	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String("pwd-windows")},
	}
	out, err := asgService.DescribeAutoScalingGroups(input)

	if err != nil {
		return nil, err
	}

	// there should always be one asg
	instances := out.AutoScalingGroups[0].Instances
	availInstances := make([]string, len(instances))

	for i, inst := range instances {
		if *inst.LifecycleState == "InService" {
			availInstances[i] = *inst.InstanceId
		}
	}

	assignedInstances, err := d.storage.InstanceGetAllWindows()
	assignedInstancesIds := []string{}
	for _, ai := range assignedInstances {
		assignedInstancesIds = append(assignedInstancesIds, ai.ID)
	}

	if err != nil {
		return nil, err
	}

	avInstanceId := d.pickFreeInstance(sessionId, availInstances, assignedInstancesIds)

	if len(avInstanceId) == 0 {
		return nil, errors.New("No Windows instance available")
	}

	iout, err := ec2Service.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(avInstanceId)},
	})
	if err != nil {
		// TODO retry x times and free the instance that was picked?
		d.releaseInstance(sessionId, avInstanceId)
		return nil, err
	}

	instance := iout.Reservations[0].Instances[0]

	instanceInfo := &instanceInfo{
		publicIP:  *instance.PublicIpAddress,
		privateIP: *instance.PrivateIpAddress,
		id:        avInstanceId,
	}

	//TODO check for free instance, ASG capacity and return

	return instanceInfo, nil
}

// select free instance and lock it into db.
// additionally check if ASG needs to be resized
func (d *windows) pickFreeInstance(sessionId string, availInstances, assignedInstances []string) string {
	for _, av := range availInstances {
		found := false
		for _, as := range assignedInstances {
			if av == as {
				found = true
				break
			}
		}

		if !found {
			err := d.storage.InstanceCreateWindows(&types.WindowsInstance{SessionId: sessionId, ID: av})
			if err != nil {
				// TODO either storage error or instance is already assigned (race condition)
			}
			return av
		}
	}
	// all availalbe instances are assigned
	return ""
}
