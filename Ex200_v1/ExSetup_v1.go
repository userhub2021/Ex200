package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
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

func cleanFstab() error {
	file, err := os.Open("/etc/fstab")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Exclude swap configurations
		if strings.Contains(line, "swap") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	output := strings.Join(lines, "\n") + "\n"
	return os.WriteFile("/etc/fstab", []byte(output), 0644)
}

func main() {
	fmt.Println("=====================================================")
	fmt.Println("   RHCSA EX200 v10 模擬試験環境構築 (ExSetup_v1.go)")
	fmt.Println("   ※ Root問題およびLVM問題は除外設定")
	fmt.Println("=====================================================")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "エラー: このスクリプトは root 権限で実行する必要があります。")
		writeExecutionLog("ExSetup_v1", "FAILED", "Error: Setup must be run as root.")
		os.Exit(1)
	}

	writeExecutionLog("ExSetup_v1", "STARTED", "Environment setup initiated.")

	// 1. Cleanup
	cleanup()

	// 2. Setup Task 2 (Network and Hostname)
	setupTask2()

	// 3. Setup Task 3 (DNF Local Repo)
	setupTask3()

	// 4. Setup Task 4 (Users, Groups, Sudo)
	setupTask4()

	// 5. Setup Task 5 (Shared Directory)
	setupTask5()

	// 6. Setup Task 6 (systemd Timer)
	setupTask6()

	// 7. Setup Task 7 (File Search & Copy)
	setupTask7()

	// 8. Setup Task 8 (Archive/Compression)
	setupTask8()

	// 9. Setup Task 9 (SELinux & Apache)
	setupTask9()

	// 10. Setup Task 12 (Swap)
	setupTask12()

	// 11. Setup Task 13 (Flatpak Package Management)
	setupTask13()

	// 12. Setup Task 14 (Shell Script)
	setupTask14()

	fmt.Println("=====================================================")
	fmt.Println("🎉 環境セットアップが正常に完了しました！")
	fmt.Println("除外した問題: 【課題 1】(root), 【課題 10】(LVM), 【課題 11】(LVM)")
	fmt.Println("対象の問題: 【課題 2〜9】, 【課題 12〜14】")
	fmt.Println("これで模擬試験問題の練習準備が整いました。")
	fmt.Println("=====================================================")
	writeExecutionLog("ExSetup_v1", "SUCCESS", "Environment setup completed successfully.")
}

func cleanup() {
	fmt.Println("\n[1/12] 既存の対象模擬試験設定・回答のクリーンアップを実行中...")

	// Task 2: Hostname cleanup
	runCmdSilent("hostnamectl", "set-hostname", "localhost")

	// Task 3: Local repo cleanup
	runCmdSilent("rm", "-f", "/etc/yum.repos.d/local.repo")

	// Task 4: User & Group cleanup
	runCmdSilent("userdel", "-r", "-f", "adminuser")
	runCmdSilent("userdel", "-r", "-f", "devuser")
	runCmdSilent("groupdel", "sysops")
	runCmdSilent("groupdel", "devs")
	runCmdSilent("rm", "-f", "/etc/sudoers.d/sysops")

	// Task 5: Shared directory cleanup
	runCmdSilent("rm", "-rf", "/common/shared_ops")

	// Task 6: Systemd Timer cleanup
	runCmdSilent("systemctl", "stop", "sys-cleanup.timer")
	runCmdSilent("systemctl", "disable", "sys-cleanup.timer")
	runCmdSilent("rm", "-f", "/etc/systemd/system/sys-cleanup.service")
	runCmdSilent("rm", "-f", "/etc/systemd/system/sys-cleanup.timer")
	runCmdSilent("systemctl", "daemon-reload")
	runCmdSilent("rm", "-f", "/var/log/sys_cleanup.log")

	// Task 7: File search cleanup
	runCmdSilent("rm", "-rf", "/root/found_files")
	runCmdSilent("rm", "-rf", "/opt/nobody_data")
	runCmdSilent("rm", "-f", "/var/tmp/system_process_nobody.log")
	runCmdSilent("rm", "-f", "/tmp/session_cache_nobody.txt")

	// Task 8: Archive backup cleanup
	runCmdSilent("rm", "-f", "/root/etc_backup.tar.gz")

	// Task 9: SELinux & Apache cleanup
	runCmdSilent("systemctl", "stop", "httpd")
	runCmdSilent("systemctl", "disable", "httpd")
	runCmdSilent("dnf", "remove", "-y", "httpd")
	runCmdSilent("rm", "-rf", "/etc/httpd")
	runCmdSilent("semanage", "port", "-d", "-t", "http_port_t", "-p", "tcp", "82")
	runCmdSilent("firewall-cmd", "--permanent", "--remove-port=82/tcp")
	runCmdSilent("firewall-cmd", "--reload")

	// Task 12: Swap cleanup
	cleanFstab()
	runCmdSilent("swapoff", "-a")
	runCmdSilent("rm", "-f", "/swapfile")
	runCmdSilent("rm", "-f", "/mock_swap")
	runCmdSilent("swapon", "-a")

	// Task 13: Flatpak cleanup
	runCmdSilent("flatpak", "uninstall", "-y", "--system", "org.gnome.peep")
	runCmdSilent("flatpak", "remote-delete", "flathub")

	// Task 14: Shell Script cleanup
	runCmdSilent("rm", "-f", "/usr/local/bin/dir_check.sh")
	runCmdSilent("rm", "-rf", "/var/tmp/script_test_env")

	fmt.Println("✔ クリーンアップが完了しました。")
}

func setupTask2() {
	fmt.Println("\n[2/12] 【課題2】ホスト名 & ネットワーク設定の準備")
	runCmdSilent("systemctl", "start", "NetworkManager")
	runCmdSilent("systemctl", "enable", "NetworkManager")
	fmt.Println("✔ NetworkManager を有効化・起動しました。")
}

func setupTask3() {
	fmt.Println("\n[3/12] 【課題3】DNFローカルリポジトリ設定の準備")
	fmt.Println("✔ ローカルリポジトリ設定の準備が完了しました。")
}

func setupTask4() {
	fmt.Println("\n[4/12] 【課題4】ユーザー・グループ・Sudo設定の準備")
	fmt.Println("✔ ユーザー・グループおよびSudo設定の準備が完了しました。")
}

func setupTask5() {
	fmt.Println("\n[5/12] 【課題5】共同作業用ディレクトリ設定の準備")
	fmt.Println("✔ 共同作業用ディレクトリ設定の準備が完了しました。")
}

func setupTask6() {
	fmt.Println("\n[6/12] 【課題6】systemd タイマー設定の準備")
	fmt.Println("✔ systemd タイマー設定の準備が完了しました。")
}

func setupTask7() {
	fmt.Println("\n[7/12] 【課題7】ファイル検索用の「nobody」所有ファイルを配置中...")
	nobodyUser, err := user.Lookup("nobody")
	if err != nil {
		fmt.Printf("エラー: nobody ユーザーが見つかりません: %v\n", err)
		return
	}
	uid, _ := strconv.Atoi(nobodyUser.Uid)
	gid, _ := strconv.Atoi(nobodyUser.Gid)

	if err := os.MkdirAll("/opt/nobody_data", 0755); err != nil {
		fmt.Printf("エラー: /opt/nobody_data の作成に失敗しました: %v\n", err)
		return
	}

	dummyFiles := []string{
		"/opt/nobody_data/secure_report.dat",
		"/var/tmp/system_process_nobody.log",
		"/tmp/session_cache_nobody.txt",
	}

	for _, file := range dummyFiles {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("エラー: ダミーファイル作成失敗 (%s): %v\n", file, err)
			continue
		}
		f.Close()
		if err := os.Chown(file, uid, gid); err != nil {
			fmt.Printf("エラー: 所有者変更失敗 (%s): %v\n", file, err)
		}
	}
	fmt.Println("✔ nobody 所有のファイルを配置しました。")
}

func setupTask8() {
	fmt.Println("\n[8/12] 【課題8】アーカイブと圧縮設定の準備")
	fmt.Println("✔ アーカイブと圧縮設定の準備が完了しました。")
}

func setupTask9() {
	fmt.Println("\n[9/12] 【課題9】SELinux/Webサービス設定の準備")
	if err := runCmdSilent("setenforce", "1"); err != nil {
		fmt.Printf("警告: SELinuxをEnforcingに設定できませんでした: %v\n", err)
	} else {
		fmt.Println("✔ SELinux を Enforcing モードに設定しました。")
	}
}

func setupTask12() {
	fmt.Println("\n[10/12] 【課題12】スワップ(Swap)領域追加の準備")
	fmt.Println("✔ スワップ設定の準備が完了しました。")
}

func setupTask13() {
	fmt.Println("\n[11/12] 【課題13】Flatpakパッケージ管理の準備")
	if _, err := exec.LookPath("flatpak"); err != nil {
		fmt.Println("flatpak コマンドが見つかりません。dnf でインストールしています...")
		if err := runCmd("dnf", "install", "-y", "flatpak"); err != nil {
			fmt.Printf("警告: flatpak のインストールに失敗しました: %v\n", err)
		} else {
			fmt.Println("✔ flatpak のインストールが完了しました。")
		}
	} else {
		fmt.Println("✔ flatpak は既にインストールされています。")
	}
}

func setupTask14() {
	fmt.Println("\n[12/12] 【課題14】シェルスクリプト検証環境の準備")
	testDir := "/var/tmp/script_test_env"
	if err := os.MkdirAll(testDir+"/sub_directory_to_ignore", 0755); err != nil {
		fmt.Printf("エラー: テスト用ディレクトリ作成失敗: %v\n", err)
		return
	}

	filesToCreate := []string{
		testDir + "/sample_file1.txt",
		testDir + "/sample_file2.log",
		testDir + "/sample_file3.conf",
		testDir + "/sub_directory_to_ignore/should_not_be_listed.txt",
	}

	for _, fPath := range filesToCreate {
		f, err := os.Create(fPath)
		if err != nil {
			fmt.Printf("エラー: テストファイル作成失敗 (%s): %v\n", fPath, err)
			continue
		}
		f.Close()
	}
	fmt.Println("✔ シェルスクリプト検証用のテスト環境を配置しました。")
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
