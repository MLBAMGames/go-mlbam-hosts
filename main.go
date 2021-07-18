package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	gh "github.com/lextoumbourou/goodhosts"
	i "github.com/tockins/interact"
)

const (
	defaultPath   = "${SystemRoot}/System32/drivers/etc/hosts"
	efaultEOL     = "\r\n"
	defaultSingle = true
)

var (
	nhltv = []string{"mf.svc.nhl.com"}
	mlbtv = []string{"playback.svcs.mlb.com", "mlb-ws-mf.media.mlb.com"}
	defaultDomains = []string{"freegamez.ga", "freesports.ddns.net"} // otherwise use domains.txt
	domains []string // domain used
)

func main() {
	printHeader()

	
	getDomains()

	i.Run(&i.Interact{
		Questions: []*i.Question{
			{
				Quest: i.Quest{
					Msg: "\n>> What you want to do ?",
					Choices: i.Choices{
						Alternatives: []i.Choice{
							{
								Text:     "Test the hosts file entries",
								Response: 1,
							},
							{
								Text:     "Add the hosts file entries",
								Response: 2,
							},
							{
								Text:     "Remove the hosts file entries",
								Response: 3,
							},
							{
								Text:     "List all hosts file entries",
								Response: 4,
							},
							{
								Text:     "Exit",
								Response: 5,
							},
						},
					},
				},
				Action: func(c i.Context) interface{} {
					todo, _ := c.Ans().Int()
					isTodoRequiresElevatedRights := todo == 2 || todo == 3

					hostsAPI := getHostsAPI()

					if isTodoRequiresElevatedRights && !hostsAPI.IsWritable() {
						printHeader()
						fmt.Fprintln(os.Stderr, "\n>> Host file not writable. Try running with elevated privileges.\n>> Right click on me and Run as Administrator")
						back(false)
					} else {
						switch todo {
						case 1:
							domain(todo)
						case 2:
							domain(todo)
						case 3:
							remove()
						case 4:
							list()
						case 5:
							os.Exit(1)
						}
					}
					return nil
				},
			},
		},
	})
}

func domain(todo int64) {
	printHeader()

	alternatives := []i.Choice{}
	for _, domain := range domains {
		alternatives = append(alternatives, i.Choice{
			Text:     domain,
			Response: domain,
		})
	}

	i.Run(&i.Interact{
		Questions: []*i.Question{
			{
				Quest: i.Quest{
					Msg: "\n>> Which domain do you want to use to replace MLB.tv/NHL.tv Authentication server?",
					Choices: i.Choices{
						Alternatives: alternatives,
					},
				},
				Action: func(c i.Context) interface{} {
					val, _ := c.Ans().String()
					ip, err := net.LookupIP(val)
					checkErr(err)

					switch todo {
					case 1:
						check(val, ip[0])
					case 2:
						add(val, ip[0])
					}
					return nil
				},
			},
		},
	})
}

func back(canGoBackToMain bool) {
	goBack := i.Choice{
		Text:     "Go back to main menu",
		Response: 1,
	}
	exit := i.Choice{
		Text:     "Exit",
		Response: 2,
	}
	alternatives := []i.Choice{exit}

	if canGoBackToMain {
		alternatives = []i.Choice{goBack, exit}
	}

	i.Run(&i.Interact{
		Questions: []*i.Question{
			{
				Quest: i.Quest{
					Msg:     "\n>> Are you done?",
					Choices: i.Choices{Alternatives: alternatives},
				},
				Action: func(c i.Context) interface{} {
					action, _ := c.Ans().Int()
					switch action {
					case 1:
						main()
					case 2:
						os.Exit(1)
					}
					return nil
				},
			},
		},
	})
}

func check(domain string, ip net.IP) {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Checking hosts entry")
	passed := true
	for _,mediaTvDomain := range append(nhltv, mlbtv...) {
		if !hostsAPI.Has(ip.String(), mediaTvDomain) {
			fmt.Println("	Failed: Cannot find", mediaTvDomain, "authentication server redirection.")
			passed = false
		}
	}

	if passed {
		fmt.Println("	Passed: Found MLB.tv/NHL.tv authentication server redirection.")
	} else {
		back(true)
	}

	fmt.Println()
	fmt.Println(">> Trying to reach MLB.tv/NHL.tv using the redirection to a custom domain")
	
	allPassed := true
	for _,mediaTvDomain := range append(nhltv, mlbtv...) {
		ips, err := net.LookupIP(mediaTvDomain)
		checkErr(err)
		fmt.Printf("	>> %v (%v) ... %v (%v)\n", mediaTvDomain, ips, domain, ip)

		passed = false
		for _, mediaTvIP := range ips {
			if ip.Equal(mediaTvIP) {
				fmt.Println("		Passed: MLB.tv/NHL.tv redirection from", mediaTvIP, "to", mediaTvDomain, "is working")
				passed = true
			}
		}
		if !passed {
			fmt.Println("		Failed: MLB.tv/NHL.tv redirection to", mediaTvDomain, "isn't working")
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("	Passed: All MLB.tv/NHL.tv redirection are working")
	} else {
		fmt.Println("	Failed: One or more MLB.tv/NHL.tv redirection are not working")
	}

	back(true)
}

func add(domain string, ip net.IP) {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Adding hosts file entries")
	for _,mediaTvDomain := range append(nhltv, mlbtv...) {
		err := hostsAPI.Add(ip.String(), mediaTvDomain)
		checkErr(err)
		fmt.Println("	Success: Added", mediaTvDomain, "redirection to", domain, "(", ip.String(), ")")
	}

	err := hostsAPI.Flush()
	checkErr(err)

	back(true)
}

func remove() {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Removing hosts entries: All references of MLB.tv/NHL.tv")
	found := 0

	for _, line := range hostsAPI.Lines {
		for _, mediaTvDomain := range append(nhltv, mlbtv...) {
			if !line.IsComment() && line.Raw != "" && itemInSlice(mediaTvDomain, line.Hosts) {
				err := hostsAPI.Remove(line.IP, mediaTvDomain)
				checkErr(err)
				fmt.Println("	Success: Removed", mediaTvDomain, "redirection to", line.IP)
				found++
			}
		}
	}

	err := hostsAPI.Flush()
	checkErr(err)

	if found == 0 {
		fmt.Println("	Nothing found, nothing done")
	}

	back(true)
}

func list() {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Listing all hosts entries")
	total := 0
	for _, line := range hostsAPI.Lines {
		var lineOutput string

		if line.IsComment() || line.Raw == "" {
			continue
		}

		lineOutput = fmt.Sprintf("	%s", line.Raw)
		if line.Err != nil {
			lineOutput = fmt.Sprintf(">> %s is malformated, it might not work!", lineOutput)
		}
		total++

		fmt.Println(lineOutput)
	}

	fmt.Println(">> Total:", total)

	back(true)
}

func itemInSlice(item string, list []string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}

	return false
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
		back(false)
	}
}

func printHeader() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
	fmt.Println("\n>> Go MLBAM hosts\n>> Windows hosts file manager to redirect NHL.tv and MLB.tv authentication to a custom hostname")
}

func getDomains() {
	if len(domains) > 0 {
		return
	}

	if _, err := os.Stat("domains.txt"); err == nil {
		file, err := os.Open("domains.txt")
		if err != nil {
			return
		}
		defer file.Close()

		fmt.Print(">> Loading domains:")
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			domain := scanner.Text()
			domains = append(domains, domain)
			fmt.Printf(" %v", domain)
		}
	} else if os.IsNotExist(err) {
		domains = defaultDomains
		return
	}
}

func getHostsAPI() gh.Hosts {
	hostsAPI, err := gh.NewHosts()
	checkErr(err)
	return hostsAPI
}
