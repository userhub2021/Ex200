package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type TestResult struct {
	Name    string
	Score   int
	Max     int
	Message string
}

func main() {
	fmt.Println("=== RHCSA EX200 V2 Answer Checker (RHEL 10) ===")
	fmt.Println("Checking configurations...")

	results := []TestResult{
		checkTask1(),
		checkTask2(),
		checkTask3(),
		checkTask4(),
		checkTask5(),
		checkTask6(),
		checkTask7(),
		checkTask8(),
		checkTask9(),
	}

	totalScore := 0
	maxScore := 0
	fmt.Println("\n--- Evaluation Results ---")
	for _, res := range results {
		fmt.Printf("[%s] Score: %d/%d\n", res.Name, res.Score, res.Max)
		if res.Message != "" {
			fmt.Printf("   Feedback: %s\n", res.Message)
		}
		totalScore += res.Score
		maxScore += res.Max
	}

	fmt.Println("\n--------------------------")
	fmt.Printf("TOTAL SCORE: %d / %d\n", totalScore, maxScore)
	if totalScore >= 70 {
		fmt.Println("Result: PASSED (70% or higher)")
	} else {
		fmt.Println("Result: FAILED (Required: 70% or higher)")
	}
}

func checkTask1() TestResult {
	res := TestResult{Name: "Task 1 (Users, Groups, & ACLs)", Score: 0, Max: 15}
	var messages []string

	// 1. Group 'sysops' exists (3 pts)
	if err := exec.Command("getent", "group", "sysops").Run(); err == nil {
		res.Score += 3
	} else {
		messages = append(messages, "Group 'sysops' does not exist.")
	}

	// 2. User 'operator' exists and has secondary group 'sysops' (3 pts)
	if out, err := exec.Command("id", "-Gn", "operator").Output(); err == nil {
		groups := strings.Fields(string(out))
		hasSysops := false
		for _, g := range groups {
			if g == "sysops" {
				hasSysops = true
				break
			}
		}
		if hasSysops {
			res.Score += 3
		} else {
			messages = append(messages, "User 'operator' is not a member of 'sysops'.")
		}
	} else {
		messages = append(messages, "User 'operator' does not exist.")
	}

	// 3. User 'sysnobody' exists and has no login shell (3 pts)
	if out, err := exec.Command("getent", "passwd", "sysnobody").Output(); err == nil {
		fields := strings.Split(strings.TrimSpace(string(out)), ":")
		if len(fields) >= 7 && (fields[6] == "/sbin/nologin" || fields[6] == "/usr/sbin/nologin") {
			res.Score += 3
		} else {
			messages = append(messages, "User 'sysnobody' login shell is not /sbin/nologin.")
		}
	} else {
		messages = append(messages, "User 'sysnobody' does not exist.")
	}

	// 4. Directory /var/share/sysops (4 pts)
	if fi, err := os.Stat("/var/share/sysops"); err == nil {
		if fi.IsDir() {
			stat := fi.Sys().(*syscall.Stat_t)
			ownerOK := (stat.Uid == 0)
			groupOK := false
			if grpOut, grpErr := exec.Command("getent", "group", "sysops").Output(); grpErr == nil {
				grpFields := strings.Split(strings.TrimSpace(string(grpOut)), ":")
				if len(grpFields) >= 3 && fmt.Sprintf("%d", stat.Gid) == grpFields[2] {
					groupOK = true
				}
			}
			mode := fi.Mode()
			isSGID := (mode & os.ModeSetgid) != 0
			perm := mode.Perm()
			permOK := (perm&0007 == 0) && (perm&0070 == 0070) // group rwx, others no access

			if ownerOK && groupOK && isSGID && permOK {
				res.Score += 4
			} else {
				messages = append(messages, "Directory '/var/share/sysops' has wrong ownership or permission. Check SGID, group access, and other users access restriction.")
			}
		} else {
			messages = append(messages, "'/var/share/sysops' is not a directory.")
		}
	} else {
		messages = append(messages, "Directory '/var/share/sysops' does not exist.")
	}

	// 5. Check ACL for guestuser (r-x) (2 pts)
	if out, err := exec.Command("getfacl", "-p", "/var/share/sysops").Output(); err == nil {
		aclStr := string(out)
		if strings.Contains(aclStr, "user:guestuser:r-x") {
			res.Score += 2
		} else {
			messages = append(messages, "ACL for 'guestuser' (r-x) is not set on '/var/share/sysops'.")
		}
	} else {
		messages = append(messages, "Failed to run getfacl on '/var/share/sysops'.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask2() TestResult {
	res := TestResult{Name: "Task 2 (Systemd Timer Schedule)", Score: 0, Max: 15}
	var messages []string

	// 1. Files existence (5 pts)
	timerFile := "/etc/systemd/system/backup.timer"
	serviceFile := "/etc/systemd/system/backup.service"
	hasTimer := false
	hasService := false
	if _, err := os.Stat(timerFile); err == nil {
		hasTimer = true
	}
	if _, err := os.Stat(serviceFile); err == nil {
		hasService = true
	}

	if hasTimer && hasService {
		res.Score += 5
	} else {
		messages = append(messages, "Systemd backup.timer or backup.service file does not exist in /etc/systemd/system/.")
	}

	// 2. Active and Enabled states (5 pts)
	timerActive := false
	timerEnabled := false
	if out, err := exec.Command("systemctl", "is-active", "backup.timer").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "active" {
			timerActive = true
		}
	}
	if out, err := exec.Command("systemctl", "is-enabled", "backup.timer").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "enabled" {
			timerEnabled = true
		}
	}

	if timerActive && timerEnabled {
		res.Score += 5
	} else {
		messages = append(messages, "backup.timer is not active (running) or not enabled (auto-start).")
	}

	// 3. Configurations: timer schedule and service command (5 pts)
	configOK := true
	if hasTimer {
		if content, err := ioutil.ReadFile(timerFile); err == nil {
			cStr := string(content)
			hasTime := strings.Contains(cStr, "03:15")
			hasDays := strings.Contains(cStr, "Mon..Fri") || strings.Contains(cStr, "Mon-Fri") || strings.Contains(cStr, "1-5")
			if !hasTime || !hasDays {
				configOK = false
				messages = append(messages, "backup.timer calendar schedule is incorrect (Expected 03:15 on Mon-Fri).")
			}
		}
	}
	if hasService {
		if content, err := ioutil.ReadFile(serviceFile); err == nil {
			cStr := string(content)
			expectedCmd := "/usr/bin/tar -czf /backup/logs_backup.tar.gz /var/log/messages"
			if !strings.Contains(cStr, expectedCmd) {
				configOK = false
				messages = append(messages, "backup.service does not call the correct tar backup command.")
			}
		}
	}

	if hasTimer && hasService && configOK {
		res.Score += 5
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask3() TestResult {
	res := TestResult{Name: "Task 3 (LVM Storage)", Score: 0, Max: 15}
	var messages []string

	// 1. PV check on /dev/sdb (3 pts)
	if out, err := exec.Command("pvs", "--noheadings", "-o", "vg_name", "/dev/sdb").Output(); err == nil {
		vg := strings.TrimSpace(string(out))
		if vg == "vg_exam" {
			res.Score += 3
		} else {
			messages = append(messages, fmt.Sprintf("PV '/dev/sdb' is assigned to VG '%s' instead of 'vg_exam'.", vg))
		}
	} else {
		messages = append(messages, "PV '/dev/sdb' is not initialized or not found in LVM.")
	}

	// 2. LV check (3 pts)
	if out, err := exec.Command("lvs", "--noheadings", "-o", "lv_size,lv_name", "vg_exam").Output(); err == nil {
		lvsStr := string(out)
		if strings.Contains(lvsStr, "lv_store") {
			if strings.Contains(strings.ToLower(lvsStr), "300") {
				res.Score += 3
			} else {
				messages = append(messages, "LV 'lv_store' found, but size is not 300MiB.")
			}
		} else {
			messages = append(messages, "LV 'lv_store' not found in VG 'vg_exam'.")
		}
	} else {
		messages = append(messages, "Failed to query LVs in VG 'vg_exam'.")
	}

	// 3. Mount and FSType check (3 pts for mount, 3 pts for xfs = 6 pts)
	if out, err := exec.Command("findmnt", "-n", "-o", "FSTYPE,TARGET", "/mnt/store").Output(); err == nil {
		info := strings.Fields(string(out))
		if len(info) >= 2 {
			fsType := info[0]
			target := info[1]
			if target == "/mnt/store" {
				if fsType == "xfs" {
					res.Score += 6
				} else {
					res.Score += 3
					messages = append(messages, fmt.Sprintf("Filesystem at /mnt/store is '%s' instead of 'xfs'.", fsType))
				}
			}
		} else {
			messages = append(messages, "/mnt/store is not mounted.")
		}
	} else {
		messages = append(messages, "/mnt/store is not mounted.")
	}

	// 4. fstab check (persistent mount) (3 pts)
	if fstabBytes, err := ioutil.ReadFile("/etc/fstab"); err == nil {
		fstabStr := string(fstabBytes)
		hasFstabEntry := false
		for _, line := range strings.Split(fstabStr, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "/mnt/store" {
				hasFstabEntry = true
				break
			}
		}
		if hasFstabEntry {
			res.Score += 3
		} else {
			messages = append(messages, "Persistent mount entry for '/mnt/store' not found in '/etc/fstab'.")
		}
	} else {
		messages = append(messages, "Could not read '/etc/fstab'.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask4() TestResult {
	res := TestResult{Name: "Task 4 (Firewall & SELinux Port)", Score: 0, Max: 15}
	var messages []string

	// 1. Firewall check (8282/tcp) (7 pts)
	if err := exec.Command("firewall-cmd", "--query-port=8282/tcp", "--permanent").Run(); err == nil {
		res.Score += 7
	} else {
		messages = append(messages, "Port 8282/tcp is not permanently allowed in firewalld.")
	}

	// 2. SELinux port check (http_port_t contains 8282) (8 pts)
	if out, err := exec.Command("semanage", "port", "-l").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		foundPort := false
		for _, line := range lines {
			if strings.HasPrefix(line, "http_port_t") {
				if strings.Contains(line, "8282") {
					foundPort = true
					break
				}
			}
		}
		if foundPort {
			res.Score += 8
		} else {
			messages = append(messages, "Port 8282/tcp is not mapped to SELinux type 'http_port_t'.")
		}
	} else {
		messages = append(messages, "Failed to run 'semanage port -l' to verify SELinux configuration.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask5() TestResult {
	res := TestResult{Name: "Task 5 (Flatpak Repository)", Score: 0, Max: 10}
	var messages []string

	// Check if 'flathub' remote is added under user level of user 'operator'
	cmd := exec.Command("sudo", "-u", "operator", "flatpak", "remotes", "--user")
	out, err := cmd.Output()
	if err != nil {
		messages = append(messages, "Failed to check flatpak remotes. Ensure flatpak is installed and running.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	outputStr := string(out)
	lines := strings.Split(outputStr, "\n")
	foundRepo := false
	foundURL := false

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "flathub" {
			foundRepo = true
			if strings.Contains(line, "https://dl.flathub.org/repo/") {
				foundURL = true
			}
			break
		}
	}

	if foundRepo {
		res.Score += 5
	} else {
		messages = append(messages, "Flatpak remote 'flathub' was not found under user level for user 'operator'.")
	}

	if foundURL {
		res.Score += 5
	} else if foundRepo {
		messages = append(messages, "Flatpak remote 'flathub' exists, but the URL is incorrect (Expected 'https://dl.flathub.org/repo/').")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask6() TestResult {
	res := TestResult{Name: "Task 6 (Systemd Management)", Score: 0, Max: 10}
	var messages []string

	// 6-1. Default Boot Target (3 pts)
	if out, err := exec.Command("systemctl", "get-default").Output(); err == nil {
		target := strings.TrimSpace(string(out))
		if target == "multi-user.target" {
			res.Score += 3
		} else {
			messages = append(messages, fmt.Sprintf("Default target is '%s' instead of 'multi-user.target'.", target))
		}
	} else {
		messages = append(messages, "Failed to query default target.")
	}

	// 6-2. Masked Service (3 pts)
	if out, err := exec.Command("systemctl", "is-enabled", "cockpit.socket").Output(); err == nil {
		status := strings.TrimSpace(string(out))
		if status == "masked" {
			res.Score += 3
		} else {
			messages = append(messages, fmt.Sprintf("cockpit.socket is enabled status '%s' instead of 'masked'.", status))
		}
	} else {
		messages = append(messages, "Failed to query state of cockpit.socket.")
	}

	// 6-3. Custom Service myscript (4 pts)
	hasService := false
	serviceFile := "/etc/systemd/system/myscript.service"
	if _, err := os.Stat(serviceFile); err == nil {
		hasService = true
	}

	if hasService {
		isActive := false
		isEnabled := false
		if out, err := exec.Command("systemctl", "is-active", "myscript.service").Output(); err == nil {
			if strings.TrimSpace(string(out)) == "active" {
				isActive = true
			}
		}
		if out, err := exec.Command("systemctl", "is-enabled", "myscript.service").Output(); err == nil {
			if strings.TrimSpace(string(out)) == "enabled" {
				isEnabled = true
			}
		}

		if isActive && isEnabled {
			res.Score += 4
		} else {
			messages = append(messages, "myscript.service is found but not active or not enabled.")
		}
	} else {
		messages = append(messages, "myscript.service configuration file not found in /etc/systemd/system/.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask7() TestResult {
	res := TestResult{Name: "Task 7 (Processes & Tuning)", Score: 0, Max: 10}
	var messages []string

	// 7-1. Tuned Active Profile (5 pts)
	if out, err := exec.Command("tuned-adm", "active").Output(); err == nil {
		output := string(out)
		if strings.Contains(output, "virtual-guest") {
			res.Score += 5
		} else {
			messages = append(messages, fmt.Sprintf("Tuned active profile does not match 'virtual-guest'. Current output: %s", strings.TrimSpace(output)))
		}
	} else {
		messages = append(messages, "Failed to query active tuned profile.")
	}

	// 7-2. Process nice value adjustment (5 pts)
	pgrepCmd := exec.Command("pgrep", "-f", "/usr/local/bin/heavy-work")
	pidBytes, err := pgrepCmd.Output()
	if err == nil {
		pids := strings.Fields(string(pidBytes))
		if len(pids) > 0 {
			pid := pids[0]
			psCmd := exec.Command("ps", "-o", "nice=", "-p", pid)
			niceBytes, psErr := psCmd.Output()
			if psErr == nil {
				niceVal := strings.TrimSpace(string(niceBytes))
				if niceVal == "10" {
					res.Score += 5
				} else {
					messages = append(messages, fmt.Sprintf("heavy-work process Nice value is '%s' instead of '10'.", niceVal))
				}
			} else {
				messages = append(messages, "Failed to fetch nice value of the running heavy-work process.")
			}
		} else {
			messages = append(messages, "heavy-work process is not running.")
		}
	} else {
		messages = append(messages, "heavy-work process is not running (PID not found).")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask8() TestResult {
	res := TestResult{Name: "Task 8 (Root Password Recovery)", Score: 0, Max: 5}
	var messages []string

	expectedPass := "RootRecoveryPass123!"

	data, err := ioutil.ReadFile("/etc/shadow")
	if err != nil {
		messages = append(messages, "Failed to read '/etc/shadow' for verification.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	lines := strings.Split(string(data), "\n")
	var rootShadow string
	for _, line := range lines {
		if strings.HasPrefix(line, "root:") {
			rootShadow = line
			break
		}
	}

	if rootShadow == "" {
		messages = append(messages, "Root user entry not found in '/etc/shadow'.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	fields := strings.Split(rootShadow, ":")
	if len(fields) < 2 {
		messages = append(messages, "Malformed root entry in '/etc/shadow'.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	hash := fields[1]
	if hash == "*" || hash == "!" || hash == "!!" || hash == "" {
		messages = append(messages, "Root password is locked, disabled, or not set.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	cmdStr := fmt.Sprintf("import crypt; print(crypt.crypt('%s', '%s'))", expectedPass, hash)
	cmd := exec.Command("python3", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		messages = append(messages, fmt.Sprintf("Python3 verification execution failed: %v", err))
		res.Message = strings.Join(messages, " ")
		return res
	}

	calculatedHash := strings.TrimSpace(string(out))
	if calculatedHash == hash {
		res.Score += 5
	} else {
		messages = append(messages, "Root password does not match 'RootRecoveryPass123!'. Check if you successfully completed the recovery procedure.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}

func checkTask9() TestResult {
	res := TestResult{Name: "Task 9 (Journalctl Log Extraction)", Score: 0, Max: 5}
	var messages []string

	filePath := "/root/journal_err.txt"

	// 1. File existence check (2 pts)
	if _, err := os.Stat(filePath); err != nil {
		messages = append(messages, "File '/root/journal_err.txt' does not exist.")
		res.Message = strings.Join(messages, " ")
		return res
	}
	res.Score += 2

	// 2. Content validation (3 pts)
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		messages = append(messages, "Failed to read '/root/journal_err.txt'.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	userContent := string(fileBytes)
	if strings.TrimSpace(userContent) == "" {
		messages = append(messages, "File is empty.")
		res.Message = strings.Join(messages, " ")
		return res
	}

	// Read current boots error logs from systemd-journald
	out, err := exec.Command("journalctl", "-b", "-p", "err", "--no-pager").Output()
	if err != nil {
		// Fallback to general err check
		out, err = exec.Command("journalctl", "-p", "err", "--no-pager").Output()
	}

	if err == nil {
		expectedLogs := string(out)
		expectedLines := strings.Split(expectedLogs, "\n")
		
		// Take a few log lines from user output and see if they exist in actual expected output
		userLines := strings.Split(userContent, "\n")
		matchCount := 0
		checkCount := 0
		
		for _, uLine := range userLines {
			uLine = strings.TrimSpace(uLine)
			if uLine == "" || strings.HasPrefix(uLine, "--") {
				continue
			}
			checkCount++
			
			// Check if this line is in the expected output (using substring matching)
			matched := false
			for _, eLine := range expectedLines {
				if strings.Contains(eLine, uLine) || strings.Contains(uLine, eLine) {
					matched = true
					break
				}
			}
			if matched {
				matchCount++
			}
		}

		// Calculate accuracy rate (if > 50% lines match, we count it as correct)
		if checkCount > 0 {
			accuracy := float64(matchCount) / float64(checkCount)
			if accuracy >= 0.5 {
				res.Score += 3
			} else {
				messages = append(messages, fmt.Sprintf("Content accuracy is low (%.1f%%). Make sure you filtered logs by '-b' (current boot) and '-p err' (error priority).", accuracy*100))
			}
		} else {
			messages = append(messages, "No valid log entries found in the file.")
		}
	} else {
		messages = append(messages, "Failed to extract current journald logs for reference check.")
	}

	res.Message = strings.Join(messages, " ")
	return res
}
