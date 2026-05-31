package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func main() {
	fmt.Println("=== RHCSA EX200 V2 Environment Setup (RHEL 10) ===")

	// check if running as root
	if os.Getuid() != 0 {
		fmt.Println("Error: This setup script must be run as root.")
		writeExecutionLog("ExSetup_v2", "FAILED", "Error: Setup must be run as root.")
		os.Exit(1)
	}

	writeExecutionLog("ExSetup_v2", "STARTED", "Environment setup V2 initiated.")

	// 1. Cleanup
	cleanup()

	// 2. Task 1 Setup
	setupTask1()

	// 3. Task 2 Setup
	setupTask2()

	// 4. Task 3 Setup
	setupTask3()

	// 5. Task 5 Setup
	setupTask5()

	// 6. Task 6 Setup
	setupTask6()

	// 7. Task 7 Setup
	setupTask7()

	// 9. Task 9 Setup (Dummy error log generation)
	setupTask9()

	fmt.Println("\n=== Setup Completed Successfully ===")
	fmt.Println("Ready for the exam. Read instructions in:")
	fmt.Println("  /root/Ex200/Ex200_v2/Ex200V2.md")
	writeExecutionLog("ExSetup_v2", "SUCCESS", "Environment setup V2 completed successfully.")
}

func cleanup() {
	fmt.Println("\n[1/9] Cleaning up previous configurations...")

	// Task 9 cleanup
	runCmdSilent("rm", "-f", "/root/journal_err.txt")

	// Task 7 cleanup (Processes & Tuned)
	runCmdSilent("pkill", "-f", "/usr/local/bin/heavy-work")
	runCmdSilent("rm", "-f", "/usr/local/bin/heavy-work")
	runCmdSilent("tuned-adm", "profile", "balanced")

	// Task 6 cleanup (Systemd Management)
	runCmdSilent("systemctl", "set-default", "graphical.target")
	runCmdSilent("systemctl", "unmask", "cockpit.socket")
	runCmdSilent("systemctl", "stop", "myscript.service")
	runCmdSilent("systemctl", "disable", "myscript.service")
	runCmdSilent("rm", "-f", "/etc/systemd/system/myscript.service")
	runCmdSilent("rm", "-f", "/usr/local/bin/myscript.sh")
	runCmdSilent("rm", "-f", "/var/log/myscript.log")

	// Task 5 (Flatpak) cleanup
	runCmdSilent("sudo", "-u", "operator", "flatpak", "remote-delete", "--user", "flathub")
	runCmdSilent("rm", "-f", "/home/operator/flathub.flatpakrepo")

	// Task 4 cleanup
	runCmdSilent("firewall-cmd", "--permanent", "--remove-port=8282/tcp")
	runCmdSilent("firewall-cmd", "--reload")
	runCmdSilent("semanage", "port", "-d", "-t", "http_port_t", "-p", "tcp", "8282")

	// Task 3 (LVM) cleanup
	runCmdSilent("umount", "/mnt/store")
	runCmdSilent("rm", "-rf", "/mnt/store")

	// Remove fstab entries
	if fstabBytes, err := ioutil.ReadFile("/etc/fstab"); err == nil {
		lines := strings.Split(string(fstabBytes), "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.Contains(line, "/mnt/store") && !strings.Contains(line, "vg_exam-lv_store") {
				newLines = append(newLines, line)
			}
		}
		ioutil.WriteFile("/etc/fstab", []byte(strings.Join(newLines, "\n")), 0644)
	}

	runCmdSilent("lvremove", "-y", "/dev/vg_exam/lv_store")
	runCmdSilent("vgremove", "-y", "vg_exam")
	runCmdSilent("pvremove", "-y", "/dev/sdb")

	// Task 2 (Systemd Timer) cleanup
	runCmdSilent("systemctl", "stop", "backup.timer")
	runCmdSilent("systemctl", "disable", "backup.timer")
	runCmdSilent("rm", "-f", "/etc/systemd/system/backup.timer")
	runCmdSilent("rm", "-f", "/etc/systemd/system/backup.service")
	runCmdSilent("rm", "-rf", "/backup")

	// Task 1 cleanup
	runCmdSilent("userdel", "-r", "-f", "operator")
	runCmdSilent("userdel", "-r", "-f", "sysnobody")
	runCmdSilent("userdel", "-r", "-f", "guestuser")
	runCmdSilent("groupdel", "sysops")
	runCmdSilent("rm", "-rf", "/var/share/sysops")

	// Reload systemd daemon after service file removals
	runCmdSilent("systemctl", "daemon-reload")
}

func setupTask1() {
	fmt.Println("\n[2/9] Setting up Task 1 (Users, Groups)...")
	if err := runCmd("groupadd", "sysops"); err != nil {
		fmt.Printf("Warning: groupadd sysops: %v\n", err)
	}
	if err := runCmd("useradd", "operator"); err != nil {
		fmt.Printf("Warning: useradd operator: %v\n", err)
	}
	runCmd("usermod", "-aG", "sysops", "operator")
	
	// Set operator password to 'password'
	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader("operator:password\n")
	cmd.Run()

	// Create user guestuser which will be used for ACL task
	if err := runCmd("useradd", "guestuser"); err != nil {
		fmt.Printf("Warning: useradd guestuser: %v\n", err)
	}
}

func setupTask2() {
	fmt.Println("\n[3/9] Setting up Task 2 (Systemd Timer Target Directory)...")
	runCmd("mkdir", "-p", "/backup")
	runCmd("chown", "operator:sysops", "/backup")
	runCmd("chmod", "770", "/backup")
	
	// Ensure /var/log/messages exists
	if _, err := os.Stat("/var/log/messages"); os.IsNotExist(err) {
		ioutil.WriteFile("/var/log/messages", []byte("Initial logs for exam\n"), 0644)
	}
}

func setupTask3() {
	fmt.Println("\n[4/9] Setting up Task 3 (LVM)...")
	fmt.Println("Note: Physical disk environment setup is skipped as requested.")
	fmt.Println("Make sure /dev/sdb is available in your test RHEL10 system.")
}

func setupTask5() {
	fmt.Println("\n[5/9] Setting up Task 5 (Flatpak config files)...")

	flatpakrepoContent := `[Flatpak Repo]
Title=Flathub
Url=https://dl.flathub.org/repo/
Homepage=https://flathub.org/
Comment=Central repository of Flatpak applications
Description=Central repository of Flatpak applications
Icon=https://dl.flathub.org/repo/logo.svg
`
	repoPath := "/home/operator/flathub.flatpakrepo"
	if err := ioutil.WriteFile(repoPath, []byte(flatpakrepoContent), 0644); err != nil {
		fmt.Printf("Error writing flatpakrepo config: %v\n", err)
		return
	}

	// Change ownership to operator
	runCmd("chown", "operator:operator", repoPath)
}

func setupTask6() {
	fmt.Println("\n[6/9] Setting up Task 6 (Custom Script for Systemd Service)...")

	scriptContent := `#!/bin/bash
echo "myscript executed at $(date)" >> /var/log/myscript.log
`
	scriptPath := "/usr/local/bin/myscript.sh"
	if err := ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		fmt.Printf("Error writing custom script: %v\n", err)
	}
}

func setupTask7() {
	fmt.Println("\n[7/9] Setting up Task 7 (Processes for tuning & nice adjustment)...")

	heavyWorkContent := `#!/bin/bash
# Simulates background work
while true; do
	sleep 60
done
`
	heavyWorkPath := "/usr/local/bin/heavy-work"
	if err := ioutil.WriteFile(heavyWorkPath, []byte(heavyWorkContent), 0755); err != nil {
		fmt.Printf("Error writing heavy-work script: %v\n", err)
		return
	}

	// Make sure tuned daemon is running
	runCmdSilent("systemctl", "start", "tuned")

	// Run heavy-work in background with nice=0
	cmd := exec.Command("nohup", "/usr/local/bin/heavy-work")
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error running heavy-work script in background: %v\n", err)
	}
}

func setupTask9() {
	fmt.Println("\n[9/9] Generating dummy system error logs for Task 9...")
	// Force systemd to log an error in journald by trying to start a non-existent service
	runCmdSilent("systemctl", "start", "non_existent_dummy_error_trigger.service")
}

func writeExecutionLog(progName, status, detail string) {
	logFile := "/var/log/ex200_execution.log"
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile = "/tmp/ex200_execution.log"
		f, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	username := "unknown"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	fmt.Fprintf(f, "[%s] [%s] User: %s | Status: %s | Details: %s\n", timestamp, progName, username, status, detail)
}
