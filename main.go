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
	nhltv   = "mf.svc.nhl.com"
	domain1 = "freegamez.ga"
	domain2 = "freesports.ddns.net"

	defaultPath   = "${SystemRoot}/System32/drivers/etc/hosts"
	efaultEOL     = "\r\n"
	defaultSingle = true
)

var (
	domains []string
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
								Text:     "Test NHL.tv redirection to NHLGames",
								Response: 1,
							},
							{
								Text:     "Add entry for NHL.tv to NHLGames",
								Response: 2,
							},
							{
								Text:     "Remove NHLGames entries",
								Response: 3,
							},
							{
								Text:     "List Windows hosts file entries",
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
						back(true)
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
					Msg: "\n>> Which NHLGames domain?",
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
						add(ip[0])
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
	if hostsAPI.Has(ip.String(), nhltv) {
		fmt.Println("	Passed: NHL.tv has a redirection to NHLGames")
	} else {
		fmt.Println("	Failed: NHL.tv has a redirection to NHLGames")
	}

	fmt.Println(">> Trying to reach NHL.tv using NHLGames server")
	nhltvIPs, err := net.LookupIP(nhltv)
	checkErr(err)
	fmt.Printf(">> %v (%v) ... %v (%v)\n", nhltv, nhltvIPs, domain, ip)

	passed := false
	for _, nhltvIP := range nhltvIPs {
		if ip.Equal(nhltvIP) {
			passed = true
		}
	}
	if passed {
		fmt.Println("	Passed: NHL.tv redirection is working")
	} else {
		fmt.Println("	Failed: NHL.tv redirection is not working")
	}

	back(true)
}

func add(ip net.IP) {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Adding hosts entry: NHL.tv to NHLGames")
	err := hostsAPI.Add(ip.String(), nhltv)
	checkErr(err)

	err = hostsAPI.Flush()
	checkErr(err)

	fmt.Println("	Success: Added", ip.String())

	back(false)
}

func remove() {
	printHeader()

	hostsAPI := getHostsAPI()

	fmt.Println("\n>> Removing hosts entries: All references of NHL.tv")
	found := 0

	for _, line := range hostsAPI.Lines {
		if !line.IsComment() && line.Raw != "" && itemInSlice(nhltv, line.Hosts) {
			err := hostsAPI.Remove(line.IP, nhltv)
			checkErr(err)
			fmt.Println("	Success: Removed", line.IP)
			found++
		}
	}

	err := hostsAPI.Flush()
	checkErr(err)

	if found == 0 {
		fmt.Println("	Nothing found, nothing done")
	}

	back(false)
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
	fmt.Println("\n>> NHL.tv to NHLGames\n>> Windows hosts file manager")
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
		domains = []string{domain1, domain2}
		return
	}
}

func getHostsAPI() gh.Hosts {
	hostsAPI, err := gh.NewHosts()
	checkErr(err)
	return hostsAPI
}
