package awxmgr

import (
	"time"

	"github.com/lz006/extended-awx-client-go/eawx"
	"github.com/lz006/monitor-operator/pkg/crdmgr"
)

func init() {

}

// Starts connection to awx and triggers polling loop
func Start(channel chan *map[string]crdmgr.HostGroup) {

	connection, err := eawx.NewConnectionBuilder().
		URL("http://192.168.178.41/api"). // URL is mandatory
		Username("admin").
		Password("password").
		Insecure(true).
		Build() // Create the client
	if err != nil {
		panic(err)
	}

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

	defer connection.Close()
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

		for i := 0; i < len(groupsArrayRef); i++ {
			tmpGroup := (*groupsArrayRef[i])
			inventory[tmpGroup] = append(inventory[tmpGroup], host)
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
