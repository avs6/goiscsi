package goiscsi

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const (
	// ChrootDirectory allows the iscsiadm commands to be run within a chrooted path, helpful for containerized services
	ChrootDirectory = "chrootDirectory"
	// DefaultInitiatorNameFile is the default file which contains the initiator names
	DefaultInitiatorNameFile = "/etc/iscsi/initiatorname.iscsi"
)

// LinuxISCSI provides many iSCSI-specific functions.
type LinuxISCSI struct {
	ISCSIType
}

// NewLinuxISCSI returns an LinuxISCSI client
func NewLinuxISCSI(opts map[string]string) *LinuxISCSI {
	var iscsi LinuxISCSI
	iscsi = LinuxISCSI{
		ISCSIType: ISCSIType{
			mock:    false,
			options: opts,
		},
	}

	return &iscsi
}

func (iscsi *LinuxISCSI) getChrootDirectory() string {
	s := iscsi.options[ChrootDirectory]
	if s == "" {
		s = "/"
	}
	return s
}

func (iscsi *LinuxISCSI) buildISCSICommand(cmd []string) []string {
	if iscsi.getChrootDirectory() == "/" {
		return cmd
	}
	command := []string{"chroot", iscsi.getChrootDirectory()}
	for _, s := range cmd {
		command = append(command, s)
	}
	return command
}

// DiscoverTargets runs an iSCSI discovery and returns a list of targets.
func (iscsi *LinuxISCSI) DiscoverTargets(address string, login bool) ([]ISCSITarget, error) {
	return iscsi.discoverTargets(address, login)
}

func (iscsi *LinuxISCSI) discoverTargets(address string, login bool) ([]ISCSITarget, error) {
	// iSCSI discovery is done via the iscsiadm cli
	// iscsiadm -m discovery -t st --portal <target>
	exe := iscsi.buildISCSICommand([]string{"iscsiadm", "-m", "discovery", "-t", "st", "--portal", address})
	cmd := exec.Command(exe[0], exe[1:]...)

	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error discovering %s: %v", address, err)
		return []ISCSITarget{}, err
	}

	targets := make([]ISCSITarget, 0)

	for _, line := range strings.Split(string(out), "\n") {
		// one line of the output should look like:
		// 10.247.73.130:3260,0 iqn.1992-04.com.emc:600009700bcbb70e3287017400000001
		// Portal,GroupTag Target
		tokens := strings.Split(line, " ")
		// make sure we got two tokens
		if len(tokens) == 2 {
			addrtag := strings.Split(line, " ")[0]
			tgt := strings.Split(line, " ")[1]
			targets = append(targets,
				ISCSITarget{
					Portal:   strings.Split(addrtag, ",")[0],
					GroupTag: strings.Split(addrtag, ",")[1],
					Target:   tgt,
				})
		}
	}
	// log into the target if asked
	if login {
		for _, t := range targets {
			iscsi.PerformLogin(t)
		}
	}

	return targets, nil
}

// GetInitiators returns a list of initiators on the local system.
func (iscsi *LinuxISCSI) GetInitiators(filename string) ([]string, error) {
	return iscsi.getInitiators(filename)
}

func (iscsi *LinuxISCSI) getInitiators(filename string) ([]string, error) {

	// a slice of filename, which might exist and define the iSCSI initiators
	initiatorConfig := []string{}
	iqns := []string{}

	if filename == "" {
		// add default filename(s) here
		// /etc/iscsi/initiatorname.iscsi is the proper file for CentOS, RedHat, Debian, Ubuntu
		if iscsi.getChrootDirectory() != "/" {
			initiatorConfig = append(initiatorConfig, iscsi.getChrootDirectory()+"/"+DefaultInitiatorNameFile)
		} else {
			initiatorConfig = append(initiatorConfig, DefaultInitiatorNameFile)
		}
	} else {
		initiatorConfig = append(initiatorConfig, filename)
	}

	// for each initiatior config file
	for _, init := range initiatorConfig {
		// make sure the file exists
		_, err := os.Stat(init)
		if err != nil {
			return []string{}, err
		}

		// get the contents of the initiator config file
		cmd := exec.Command("cat", init)

		out, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error gathering initiator names: %v", err)
			return nil, err
		}
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			// remove all whitespace to catch different formatting
			l = strings.Join(strings.Fields(l), "")
			if strings.HasPrefix(l, "InitiatorName=") {
				iqns = append(iqns, strings.Split(l, "=")[1])
			}
		}
	}

	return iqns, nil
}

// PerformLogin will attempt to log into an iSCSI target
func (iscsi *LinuxISCSI) PerformLogin(target ISCSITarget) error {
	return iscsi.performLogin(target)
}

func (iscsi *LinuxISCSI) performLogin(target ISCSITarget) error {
	// iSCSI login is done via the iscsiadm cli
	// iscsiadm -m node -T <target> --portal <address> -l
	exe := iscsi.buildISCSICommand([]string{"iscsiadm", "-m", "node", "-T", target.Target, "--portal", target.Portal, "-l"})
	cmd := exec.Command(exe[0], exe[1:]...)

	_, err := cmd.Output()

	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// iscsiadm exited with an exit code != 0
			iscsiResult := -1
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				iscsiResult = status.ExitStatus()
			}
			if iscsiResult == 15 {
				// session already exists
				// do not treat this as a failure
				err = nil
			} else {
				fmt.Printf("iscsiadm login failure: %v", err)
			}
		} else {
			fmt.Printf("Error logging %s at %s: %v", target.Target, target.Portal, err)
		}

		if err != nil {
			fmt.Printf("Error logging %s at %s: %v", target.Target, target.Portal, err)
			return err
		}
	}

	return nil
}

// PerformLogout will attempt to log out of an iSCSI target
func (iscsi *LinuxISCSI) PerformLogout(target ISCSITarget) error {
	return iscsi.performLogout(target)
}

func (iscsi *LinuxISCSI) performLogout(target ISCSITarget) error {
	// iSCSI login is done via the iscsiadm cli
	// iscsiadm -m node -T <target> --portal <address> -l
	exe := iscsi.buildISCSICommand([]string{"iscsiadm", "-m", "node", "-T", target.Target, "--portal", target.Portal, "--logout"})
	cmd := exec.Command(exe[0], exe[1:]...)

	_, err := cmd.Output()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// iscsiadm exited with an exit code != 0
			iscsiResult := -1
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				iscsiResult = status.ExitStatus()
			}
			if iscsiResult == 15 {
				// session already exists
				// do not treat this as a failure
				err = nil
			} else {
				fmt.Printf("iscsiadm login failure: %v", err)
			}
		} else {
			fmt.Printf("Error logging %s at %s: %v", target.Target, target.Portal, err)
		}

		if err != nil {
			fmt.Printf("Error logging %s at %s: %v", target.Target, target.Portal, err)
			return err
		}
	}

	return nil
}

// PerformRescan will will rescan targets known to current sessions
func (iscsi *LinuxISCSI) PerformRescan() error {
	return iscsi.performRescan()
}

func (iscsi *LinuxISCSI) performRescan() error {
	exe := iscsi.buildISCSICommand([]string{"iscsiadm", "-m", "node", "--rescan"})
	cmd := exec.Command(exe[0], exe[1:]...)

	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}