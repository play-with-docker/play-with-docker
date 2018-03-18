package provisioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"

	"golang.org/x/net/websocket"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
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
	winfo, err := d.getWindowsInstanceInfo(session.Id)

	if err != nil {
		return nil, err
	}
	labels := map[string]string{
		"io.tutorius.networkid":            session.Id,
		"io.tutorius.networking.remote.ip": winfo.privateIP,
	}
	instanceName := fmt.Sprintf("%s_%s", session.Id[:8], winfo.id)

	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		d.releaseInstance(winfo.id)
		return nil, err
	}
	if err = dockerClient.ConfigCreate(instanceName, labels, []byte(instanceName)); err != nil {
		d.releaseInstance(winfo.id)
		return nil, err
	}

	instance := &types.Instance{}
	instance.Name = instanceName
	instance.Image = ""
	instance.IP = winfo.privateIP
	instance.RoutableIP = instance.IP
	instance.SessionId = session.Id
	instance.WindowsId = winfo.id
	instance.Cert = conf.Cert
	instance.Key = conf.Key
	instance.Type = conf.Type
	instance.ServerCert = conf.ServerCert
	instance.ServerKey = conf.ServerKey
	instance.CACert = conf.CACert
	instance.Tls = conf.Tls
	instance.ProxyHost = router.EncodeHost(session.Id, instance.RoutableIP, router.HostOpts{})
	instance.SessionHost = session.Host

	return instance, nil

}

func (d *windows) InstanceDelete(session *types.Session, instance *types.Instance) error {
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return err
	}

	_, err = asgService.DetachInstances(&autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String("pwd-windows"),
		InstanceIds:                    []*string{aws.String(instance.WindowsId)},
		ShouldDecrementDesiredCapacity: aws.Bool(false),
	})

	if err != nil {
		return err
	}

	//return error and don't do anything else
	if _, err := ec2Service.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{aws.String(instance.WindowsId)}}); err != nil {
		return err
	}

	err = dockerClient.ConfigDelete(instance.Name)
	if err != nil {
		return err
	}

	return d.releaseInstance(instance.WindowsId)
}

type execRes struct {
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func (d *windows) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	execBody := struct {
		Cmd []string `json:"cmd"`
	}{Cmd: cmd}

	b, err := json.Marshal(execBody)
	if err != nil {
		return -1, err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s:222/exec", instance.IP), "application/json", bytes.NewReader(b))
	if err != nil {
		log.Println(err)
		return -1, err
	}
	if resp.StatusCode != 200 {
		log.Printf("Error exec on instance %s. Got %d\n", instance.Name, resp.StatusCode)
		return -1, fmt.Errorf("Error exec on instance %s. Got %d\n", instance.Name, resp.StatusCode)
	}
	var ex execRes
	err = json.NewDecoder(resp.Body).Decode(&ex)
	if err != nil {
		return -1, err
	}
	return ex.ExitCode, nil
}

func (d *windows) InstanceFSTree(instance *types.Instance) (io.Reader, error) {
	//TODO implement
	return nil, nil
}
func (d *windows) InstanceFile(instance *types.Instance, filePath string) (io.Reader, error) {
	//TODO implement
	return nil, nil
}

func (d *windows) releaseInstance(instanceId string) error {
	return d.storage.WindowsInstanceDelete(instanceId)
}

func (d *windows) InstanceResizeTerminal(instance *types.Instance, rows, cols uint) error {
	resp, err := http.Post(fmt.Sprintf("http://%s:222/terminals/1/size?cols=%d&rows=%d", instance.IP, cols, rows), "application/json", nil)
	if err != nil {
		log.Println(err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Error resizing terminal of instance %s. Got %d\n", instance.Name, resp.StatusCode)
		return fmt.Errorf("Error resizing terminal got %d\n", resp.StatusCode)
	}
	return nil
}

func (d *windows) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	resp, err := http.Post(fmt.Sprintf("http://%s:222/terminals/1", instance.IP), "application/json", nil)
	if err != nil {
		log.Printf("Error creating terminal for instance %s. Got %v\n", instance.Name, err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Printf("Error creating terminal for instance %s. Got %d\n", instance.Name, resp.StatusCode)
		return nil, fmt.Errorf("Creating terminal got %d\n", resp.StatusCode)
	}
	url := fmt.Sprintf("ws://%s:222/terminals/1", instance.IP)
	ws, err := websocket.Dial(url, "", url)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ws, nil
}

func (d *windows) InstanceUploadFromUrl(instance *types.Instance, fileName, dest, u string) error {
	log.Printf("Downloading file [%s]\n", u)
	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("Could not download file [%s]. Error: %s\n", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Could not download file [%s]. Status code: %d\n", u, resp.StatusCode)
	}
	uploadResp, err := http.Post(fmt.Sprintf("http://%s:222/terminals/1/uploads?dest=%s&file_name=%s", instance.IP, url.QueryEscape(dest), url.QueryEscape(fileName)), "", resp.Body)
	if err != nil {
		return err
	}
	if uploadResp.StatusCode != 200 {
		return fmt.Errorf("Could not upload file [%s]. Status code: %d\n", fileName, uploadResp.StatusCode)
	}

	return nil
}

func (d *windows) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	uploadResp, err := http.Post(fmt.Sprintf("http://%s:222/terminals/1/uploads?dest=%s&file_name=%s", instance.IP, url.QueryEscape(dest), url.QueryEscape(fileName)), "", reader)
	if err != nil {
		return err
	}
	if uploadResp.StatusCode != 200 {
		return fmt.Errorf("Could not upload file [%s]. Status code: %d\n", fileName, uploadResp.StatusCode)
	}

	return nil
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

	// reverse order so older instances are first served
	sort.Sort(sort.Reverse(sort.StringSlice(availInstances)))

	for i, inst := range instances {
		if *inst.LifecycleState == "InService" {
			availInstances[i] = *inst.InstanceId
		}
	}

	assignedInstances, err := d.storage.WindowsInstanceGetAll()
	assignedInstancesIds := []string{}
	for _, ai := range assignedInstances {
		assignedInstancesIds = append(assignedInstancesIds, ai.Id)
	}

	if err != nil {
		return nil, err
	}

	avInstanceId := d.pickFreeInstance(sessionId, availInstances, assignedInstancesIds)

	if len(avInstanceId) == 0 {
		return nil, OutOfCapacityError
	}

	iout, err := ec2Service.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(avInstanceId)},
	})
	if err != nil {
		// TODO retry x times and free the instance that was picked?
		d.releaseInstance(avInstanceId)
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
			err := d.storage.WindowsInstancePut(&types.WindowsInstance{SessionId: sessionId, Id: av})
			if err != nil {
				// TODO either storage error or instance is already assigned (race condition)
			}
			return av
		}
	}
	// all availalbe instances are assigned
	return ""
}
