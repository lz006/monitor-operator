package awxmgr

import (
	"runtime/debug"
	"time"

	"github.com/lz006/extended-awx-client-go/eawx"
	"github.com/lz006/monitor-operator/pkg/crdmgr"
	"github.com/prometheus/common/log"
)

func init() {

}

// Starts connection to awx and triggers polling loop
func Start(channel chan *map[string]crdmgr.HostGroup) {

	// Connect to AWX
	log.Info("Attempt to connect to AWX...")
	connection := &eawx.Connection{}

	for true {
		err := buildConnection(connection)
		if err == nil {
			log.Info("Connection to AWX established")
			break
		} else {
			log.Error("Could not get connect to AWX. Next attempt in 30 seconds!")
			time.Sleep(time.Duration(30000 * 1000000))
		}
	}

	// Error Handling:
	// Catch possible panics caused by awx communications etc. & restart the function
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()

			err := r.(error)
			log.Error("Something went wrong. Restarting in 30 seconds! This is caused by: " + err.Error())
			time.Sleep(time.Duration(30000 * 1000000))
			go Start(channel)
		}
	}()

	// Close connection when function returns
	defer connection.Close()

	// Start infinite loop to sync AWX inventory to HostGroup-Instances in k8s/openshift
	for true {
		time.Sleep(time.Duration(5000 * 1000000))

		// poll hosts from awx including host vars
		hosts := pollHosts(connection)

		// build map[Group][]*Host that mimics the structure of an ansible inventory file
		inventory := buildInventoryMap(hosts)

		// poll groups from awx including group vars
		groups := pollGroups(connection)

		// Merge groups and hosts to HostGroup
		hostGroup := mergeToHostGroup(groups, inventory)

		// trigger crd management
		channel <- hostGroup

	}

}

func buildConnection(connection *eawx.Connection) error {

	con, err := eawx.NewConnectionBuilder().
		URL("http://192.168.178.41/api"). // URL is mandatory
		Username("admin").
		Password("password").
		Insecure(true).
		Build() // Create the client

	(*connection) = (*con)

	return err
}

func pollHosts(connection *eawx.Connection) []*eawx.Host {

	hostsResource := connection.Hosts()
	// Get a list of all Hosts.
	getHostsRequest := hostsResource.Get()
	getHostsResponse, err := getHostsRequest.Filter("host_filter", "inventory__name=\"Demo Inventory\"").Send()
	if err != nil {
		panic(err)
	}

	hosts := getHostsResponse.Results()

	return hosts
}

func buildInventoryMap(hosts []*eawx.Host) *map[string][]*eawx.Host {
	inventory := make(map[string][]*eawx.Host, len(hosts))

	for _, host := range hosts {

		groupsArrayRef := host.GroupsArray()

		// Loop through all groups of a host and build a map of it
		for i := 0; i < len(groupsArrayRef); i++ {
			tmpGroup := (*groupsArrayRef[i])

			// Check if there is a valid group name related to host
			if (*groupsArrayRef[i]) != "" {
				inventory[tmpGroup] = append(inventory[tmpGroup], host)
			} else {
				log.Info("Host skipped due to empty group name - Please check AWX inventory for Host: \"" + host.Name() + "\"")
			}
		}

	}

	return &inventory
}

func pollGroups(connection *eawx.Connection) []*eawx.Group {
	groupsResource := connection.Groups()

	// Get a list of all Groups.
	getGroupsRequest := groupsResource.Get()
	getGroupsResponse, err := getGroupsRequest.Filter("inventory__name", "Demo Inventory").Send()
	if err != nil {
		panic(err)
	}

	// Build map from result
	groups := getGroupsResponse.Results()

	return groups

}

func mergeToHostGroup(groups []*eawx.Group, hosts *map[string][]*eawx.Host) *map[string]crdmgr.HostGroup {
	result := make(map[string]crdmgr.HostGroup)

	for _, group := range groups {
		hostGroup := HostGroup{group: group, hosts: (*hosts)[group.Name()]}
		result[group.Name()] = &hostGroup
	}

	return &result

}
