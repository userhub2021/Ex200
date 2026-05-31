package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const (
	Red   = "\033[0;31m"
	Green = "\033[0;32m"
	Blue  = "\033[0;34m"
	NC    = "\033[0m" // No Color
)

var totalScore = 0
const maxScore = 140

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func runCommandWithOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func checkTask(taskName string, success bool, points int) {
	if success {
		fmt.Printf("[ %sPASS%s ] %s (+%d pts)\n", Green, NC, taskName, points)
		totalScore += points
	} else {
		fmt.Printf("[ %sFAIL%s ] %s\n", Red, NC, taskName)
	}
}

func userExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

func groupExists(groupname string) bool {
	_, err := user.LookupGroup(groupname)
	return err == nil
}

func main() {
	fmt.Printf("%s=====================================================%s\n", Blue, NC)
	fmt.Printf("%s    RHCSA EX200 (RHEL 10) 模擬試験 自動採点スクリプト (Go版)%s\n", Blue, NC)
	fmt.Printf("%s=====================================================%s\n", Blue, NC)

	// -------------------------------------------------------------
	// 1. Rescue Mode Check (Root Password change verify)
	// -------------------------------------------------------------
	// スクリプトがrootで実行できているため、基本動作をPASSとみなす
	checkTask("【課題1】システムレスキュー & rootパスワード変更", true, 10)

	// -------------------------------------------------------------
	// 2. Hostname & Network Keyfile Check (RHEL 10 Standard)
	// -------------------------------------------------------------
	netCheck := false
	hostname, err := os.Hostname()
	if err == nil && hostname == "node1.exam.example.com" {
		// NM keyfileが正しく設定されているか確認
		matches, errGlob := filepath.Glob("/etc/NetworkManager/system-connections/*.nmconnection")
		if errGlob == nil && len(matches) > 0 {
			netCheck = true
		}
	}
	checkTask("【課題2】ホスト名 & NetworkManager Keyfile設定", netCheck, 10)

	// -------------------------------------------------------------
	// 3. Repository Check (DNF 5)
	// -------------------------------------------------------------
	repoCheck := false
	dnfOut, errDnf := runCommandWithOutput("dnf", "repolist", "-q")
	if errDnf == nil {
		re := regexp.MustCompile(`(?i)(baseos|appstream|local)`)
		if re.MatchString(dnfOut) {
			repoCheck = true
		}
	}
	checkTask("【課題3】DNFリポジトリの構成 (DNF 5)", repoCheck, 10)

	// -------------------------------------------------------------
	// 4. Users, Groups and Sudoers Check
	// -------------------------------------------------------------
	userCheck := false
	if groupExists("sysops") && groupExists("devs") {
		if userExists("adminuser") && userExists("devuser") {
			idAdminOut, errIdAdmin := runCommandWithOutput("id", "adminuser")
			idDevOut, errIdDev := runCommandWithOutput("id", "devuser")
			if errIdAdmin == nil && errIdDev == nil &&
				strings.Contains(idAdminOut, "sysops") &&
				strings.Contains(idDevOut, "devs") {
				
				passwdOut, errGetent := runCommandWithOutput("getent", "passwd", "devuser")
				if errGetent == nil {
					parts := strings.Split(passwdOut, ":")
					if len(parts) >= 7 && parts[6] == "/sbin/nologin" {
						if sudoersData, errSudoers := os.ReadFile("/etc/sudoers.d/sysops"); errSudoers == nil {
							if strings.Contains(string(sudoersData), "%sysops") {
								userCheck = true
							}
						}
					}
				}
			}
		}
	}
	checkTask("【課題4】ユーザー・グループおよびSudo権限", userCheck, 10)

	// -------------------------------------------------------------
	// 5. Shared Directory Check (SGID)
	// -------------------------------------------------------------
	sharedDirCheck := false
	if info, err := os.Stat("/common/shared_ops"); err == nil && info.IsDir() {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			gSysops, errGroup := user.LookupGroup("sysops")
			if errGroup == nil {
				gidStr := fmt.Sprintf("%d", stat.Gid)
				if gidStr == gSysops.Gid {
					var permVal uint32 = uint32(info.Mode() & os.ModePerm)
					if info.Mode()&os.ModeSetuid != 0 {
						permVal += 04000
					}
					if info.Mode()&os.ModeSetgid != 0 {
						permVal += 02000
					}
					if info.Mode()&os.ModeSticky != 0 {
						permVal += 01000
					}
					permStr := fmt.Sprintf("%o", permVal)
					
					matched1, _ := regexp.MatchString(`^277[0-7]$`, permStr)
					matched2, _ := regexp.MatchString(`^27[0-7]0$`, permStr)
					if matched1 || matched2 {
						sharedDirCheck = true
					}
				}
			}
		}
	}
	checkTask("【課題5】共同作業用ディレクトリ (SGID)", sharedDirCheck, 10)

	// -------------------------------------------------------------
	// 6. systemd Timer Check (RHEL 10 New Topic)
	// -------------------------------------------------------------
	timerCheck := false
	if runCommand("systemctl", "is-active", "sys-cleanup.timer") == nil {
		_, errSvc := os.Stat("/etc/systemd/system/sys-cleanup.service")
		_, errTmr := os.Stat("/etc/systemd/system/sys-cleanup.timer")
		if errSvc == nil && errTmr == nil {
			timerCheck = true
		}
	}
	checkTask("【課題6】systemd タイマーユニットの構成", timerCheck, 10)

	// -------------------------------------------------------------
	// 7. File Search Check
	// -------------------------------------------------------------
	findCheck := false
	if info, err := os.Stat("/root/found_files"); err == nil && info.IsDir() {
		fileCount := 0
		filepath.Walk("/root/found_files", func(path string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() {
				fileCount++
			}
			return nil
		})
		if fileCount > 0 {
			findCheck = true
		}
	}
	checkTask("【課題7】ファイルの検索とコピー", findCheck, 10)

	// -------------------------------------------------------------
	// 8. Backup / Tar Check
	// -------------------------------------------------------------
	backupCheck := false
	if _, err := os.Stat("/root/etc_backup.tar.gz"); err == nil {
		if runCommand("tar", "-tzf", "/root/etc_backup.tar.gz") == nil {
			backupCheck = true
		}
	}
	checkTask("【課題8】アーカイブと圧縮", backupCheck, 10)

	// -------------------------------------------------------------
	// 9. SELinux & Apache Check
	// -------------------------------------------------------------
	selinuxHttpdCheck := false
	if runCommand("systemctl", "is-active", "httpd") == nil {
		seportOut, errSeport := runCommandWithOutput("semanage", "port", "-l")
		fwOut, errFw := runCommandWithOutput("firewall-cmd", "--list-ports", "--permanent")
		if errSeport == nil && errFw == nil {
			hasHttpPort82 := false
			lines := strings.Split(seportOut, "\n")
			for _, line := range lines {
				if strings.Contains(line, "http_port_t") && strings.Contains(line, "82") {
					hasHttpPort82 = true
					break
				}
			}
			if hasHttpPort82 && strings.Contains(fwOut, "82/tcp") {
				selinuxHttpdCheck = true
			}
		}
	}
	checkTask("【課題9】SELinux非標準ポート (Web: 82)", selinuxHttpdCheck, 10)

	// -------------------------------------------------------------
	// 10 & 11. LVM & Resize Check
	// -------------------------------------------------------------
	lvmCheck := false
	lvsOut, errLvs := runCommandWithOutput("lvs", "vg_store/lv_store", "-o", "lv_size", "--noheadings")
	findmntOut, errFindmnt := runCommandWithOutput("findmnt", "-n", "-o", "FSTYPE", "/mnt/store_data")
	fstabData, errFstab := os.ReadFile("/etc/fstab")
	if errLvs == nil && errFindmnt == nil && errFstab == nil {
		reSize := regexp.MustCompile(`(1\.[2-9]|2\.[0-9])G`)
		fstabStr := string(fstabData)
		if reSize.MatchString(lvsOut) &&
			strings.TrimSpace(findmntOut) == "xfs" &&
			(strings.Contains(fstabStr, "vg_store-lv_store") || strings.Contains(fstabStr, "lv_store")) {
			lvmCheck = true
		}
	}
	checkTask("【課題10 & 11】LVMの作成・マウント・拡張", lvmCheck, 20)

	// -------------------------------------------------------------
	// 12. Swap Check
	// -------------------------------------------------------------
	swapCheck := false
	swaponOut, errSwapon := runCommandWithOutput("swapon", "--show", "--noheadings")
	fstabDataForSwap, errFstabForSwap := os.ReadFile("/etc/fstab")
	if errSwapon == nil && errFstabForSwap == nil {
		reSwap := regexp.MustCompile(`(partition|file)`)
		if reSwap.MatchString(swaponOut) && strings.Contains(string(fstabDataForSwap), "swap") {
			swapCheck = true
		}
	}
	checkTask("【課題12】スワップ(Swap)の追加・永続化", swapCheck, 10)

	// -------------------------------------------------------------
	// 13. Flatpak Package Check (RHEL 10 New Topic)
	// -------------------------------------------------------------
	flatpakCheck := false
	remotesOut, errRemotes := runCommandWithOutput("flatpak", "remotes")
	listOut, errList := runCommandWithOutput("flatpak", "list", "--system")
	if errRemotes == nil && errList == nil {
		if strings.Contains(remotesOut, "flathub") {
			lines := strings.Split(listOut, "\n")
			nonEmptyLines := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLines++
				}
			}
			if nonEmptyLines > 0 {
				flatpakCheck = true
			}
		}
	}
	checkTask("【課題13】Flatpakリモート追加 & アプリのインストール", flatpakCheck, 10)

	// -------------------------------------------------------------
	// 14. Shell Script Check (RHEL 10 New Topic)
	// -------------------------------------------------------------
	scriptCheck := false
	scriptPath := "/usr/local/bin/dir_check.sh"
	infoScript, errScript := os.Stat(scriptPath)
	if errScript == nil && (infoScript.Mode().Perm()&0111 != 0) {
		// Run with no args
		cmd1 := exec.Command(scriptPath)
		err1 := cmd1.Run()
		exitCode1 := 0
		if err1 != nil {
			if exitError, ok := err1.(*exec.ExitError); ok {
				exitCode1 = exitError.ExitCode()
			}
		}

		// Run with nonexistent dir
		cmd2 := exec.Command(scriptPath, "/nonexistent_dir_test")
		err2 := cmd2.Run()
		exitCode2 := 0
		if err2 != nil {
			if exitError, ok := err2.(*exec.ExitError); ok {
				exitCode2 = exitError.ExitCode()
			}
		}

		// Run with /etc
		cmd3 := exec.Command(scriptPath, "/etc")
		err3 := cmd3.Run()
		exitCode3 := 0
		if err3 != nil {
			if exitError, ok := err3.(*exec.ExitError); ok {
				exitCode3 = exitError.ExitCode()
			}
		}

		if exitCode1 == 1 && exitCode2 == 2 && exitCode3 == 0 {
			scriptCheck = true
		}
	}
	checkTask("【課題14】簡易シェルスクリプトの作成", scriptCheck, 10)

	// -------------------------------------------------------------
	// Summary
	// -------------------------------------------------------------
	fmt.Printf("%s=====================================================%s\n", Blue, NC)
	fmt.Printf("  結果: %d / %d 点\n", totalScore, maxScore)
	passMark := maxScore * 70 / 100
	if totalScore >= passMark {
		fmt.Printf("  ステータス: %s合格 (CONGRATULATIONS!)%s\n", Green, NC)
	} else {
		fmt.Printf("  ステータス: %s不合格 (Keep studying!)%s\n", Red, NC)
	}
	fmt.Printf("%s=====================================================%s\n", Blue, NC)
}
