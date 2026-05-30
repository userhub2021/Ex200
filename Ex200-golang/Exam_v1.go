package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

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

func userExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

func groupExists(groupname string) bool {
	_, err := user.LookupGroup(groupname)
	return err == nil
}

func checkNobodyOwner(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	nobodyUser, err := user.Lookup("nobody")
	if err != nil {
		return false
	}
	return fmt.Sprintf("%d", stat.Uid) == nobodyUser.Uid
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
		if strings.Contains(line, "vg_store-lv_store") || strings.Contains(line, "lv_store") || strings.Contains(line, "swap") {
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
	// 実行ユーザーのチェック
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "エラー: このスクリプトは root 権限で実行する必要があります。")
		os.Exit(1)
	}

	fmt.Println("=====================================================")
	fmt.Println("   RHCSA EX200 v10 模擬試験環境の構築を開始します (Go版)")
	fmt.Println("=====================================================")

	// -------------------------------------------------------------
	// 1. クリーンアップが必要かどうかの事前チェック
	// -------------------------------------------------------------
	fmt.Println("システム上の古い模擬試験環境を検出しています...")

	needCleanup := false

	// マウントのチェック
	if runCommand("mountpoint", "-q", "/mnt/store_data") == nil {
		needCleanup = true
	}

	// LVM のチェック
	if runCommand("lvs", "vg_store") == nil ||
		runCommand("vgs", "vg_store") == nil ||
		runCommand("pvs", "/dev/vdb") == nil ||
		runCommand("pvs", "/dev/sdb") == nil {
		needCleanup = true
	}

	// ループバックイメージおよびシンボリックリンクのチェック
	if _, err := os.Stat("/var/lib/mock_extra_disk.img"); err == nil {
		needCleanup = true
	}
	if _, err := os.Lstat("/dev/vdb"); err == nil {
		needCleanup = true
	}

	// ユーザー・グループのチェック
	if userExists("adminuser") || userExists("devuser") {
		needCleanup = true
	}
	if groupExists("sysops") || groupExists("devs") {
		needCleanup = true
	}

	// 設定ファイルのチェック
	if _, err := os.Stat("/etc/sudoers.d/sysops"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/common/shared_ops"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/mnt/store_data"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/root/found_files"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/root/etc_backup.tar.gz"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/usr/local/bin/dir_check.sh"); err == nil {
		needCleanup = true
	}

	// systemd タイマーおよびサービスのチェック
	if _, err := os.Stat("/etc/systemd/system/sys-cleanup.service"); err == nil {
		needCleanup = true
	}
	if _, err := os.Stat("/etc/systemd/system/sys-cleanup.timer"); err == nil {
		needCleanup = true
	}
	if runCommand("systemctl", "is-active", "sys-cleanup.timer") == nil {
		needCleanup = true
	}

	// fstab 内の記述チェック
	if fstabData, err := os.ReadFile("/etc/fstab"); err == nil {
		fstabStr := string(fstabData)
		if strings.Contains(fstabStr, "vg_store") || strings.Contains(fstabStr, "lv_store") || strings.Contains(fstabStr, "swap") {
			needCleanup = true
		}
	}

	// ホスト名のチェック
	if currentHostname, err := os.Hostname(); err == nil {
		if currentHostname != "localhost.localdomain" && currentHostname != "localhost" {
			needCleanup = true
		}
	}

	// -------------------------------------------------------------
	// 2. クリーンアップの実行またはスキップ
	// -------------------------------------------------------------
	if needCleanup {
		fmt.Println("[1/4] 既存の模擬試験設定・回答のクリーンアップを実行中...")

		// ディレクトリマウントの解除とLVMの削除
		runCommand("umount", "/mnt/store_data")
		runCommand("lvremove", "-y", "/dev/vg_store/lv_store")
		runCommand("vgremove", "-y", "vg_store")
		runCommand("pvremove", "-y", "/dev/vdb")
		runCommand("pvremove", "-y", "/dev/sdb")

		// ループバックディスクのクリーンアップ
		if _, err := os.Stat("/var/lib/mock_extra_disk.img"); err == nil {
			loopDevsOut, err := runCommandWithOutput("losetup", "-j", "/var/lib/mock_extra_disk.img")
			if err == nil && loopDevsOut != "" {
				lines := strings.Split(loopDevsOut, "\n")
				for _, line := range lines {
					parts := strings.Split(line, ":")
					if len(parts) > 0 {
						loopDev := strings.TrimSpace(parts[0])
						if strings.HasPrefix(loopDev, "/dev/loop") {
							runCommand("losetup", "-d", loopDev)
						}
					}
				}
			}
			os.Remove("/var/lib/mock_extra_disk.img")
		}
		os.Remove("/dev/vdb")

		// ユーザー・グループの削除
		runCommand("userdel", "-r", "adminuser")
		runCommand("userdel", "-r", "devuser")
		runCommand("groupdel", "sysops")
		runCommand("groupdel", "devs")
		os.Remove("/etc/sudoers.d/sysops")

		// 各種作成ファイルの削除
		os.RemoveAll("/common/shared_ops")
		os.RemoveAll("/mnt/store_data")
		os.RemoveAll("/root/found_files")
		os.Remove("/root/etc_backup.tar.gz")
		os.Remove("/usr/local/bin/dir_check.sh")

		// systemd タイマーの停止・削除
		runCommand("systemctl", "stop", "sys-cleanup.timer")
		runCommand("systemctl", "disable", "sys-cleanup.timer")
		os.Remove("/etc/systemd/system/sys-cleanup.service")
		os.Remove("/etc/systemd/system/sys-cleanup.timer")
		runCommand("systemctl", "daemon-reload")

		// fstab から試験で追加された設定行を削除
		cleanFstab()
		runCommand("swapoff", "-a")
		os.Remove("/swapfile")
		os.Remove("/mock_swap")
		runCommand("swapon", "-a")

		// ホスト名の初期化
		runCommand("hostnamectl", "set-hostname", "localhost.localdomain")

		fmt.Println("✔ クリーンアップが完了しました。")
	} else {
		fmt.Println("[1/4] クリーンアップの対象となる不要な設定やファイルは検出されませんでした。スキップします。")
	}

	// -------------------------------------------------------------
	// 3. 追加ディスクのシミュレーション (5GB)
	// -------------------------------------------------------------
	fmt.Println("[2/4] 追加ディスク (/dev/vdb 5GB) のシミュレーションを設定中...")

	if _, err := os.Stat("/var/lib/mock_extra_disk.img"); os.IsNotExist(err) {
		file, err := os.Create("/var/lib/mock_extra_disk.img")
		if err != nil {
			fmt.Printf("エラー: イメージファイルの作成に失敗しました: %v\n", err)
			os.Exit(1)
		}
		if err := file.Truncate(5 * 1024 * 1024 * 1024); err != nil {
			file.Close()
			fmt.Printf("エラー: イメージファイルの拡張に失敗しました: %v\n", err)
			os.Exit(1)
		}
		file.Close()

		freeLoop, err := runCommandWithOutput("losetup", "-f")
		if err != nil || freeLoop == "" {
			fmt.Println("エラー: 利用可能なループバックデバイスが見つかりません。")
			os.Exit(1)
		}

		if err := runCommand("losetup", freeLoop, "/var/lib/mock_extra_disk.img"); err != nil {
			fmt.Printf("エラー: ループバックデバイスの関連付けに失敗しました: %v\n", err)
			os.Exit(1)
		}

		os.Remove("/dev/vdb")
		if err := os.Symlink(freeLoop, "/dev/vdb"); err != nil {
			fmt.Printf("エラー: シンボリックリンクの作成に失敗しました: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✔ 仮想ディスク /dev/vdb を正常に配置しました（実体: %s）\n", freeLoop)
	} else {
		fmt.Println("✔ 仮想ディスク /dev/vdb は既に配置されています。スキップします。")
	}

	// -------------------------------------------------------------
	// 4. 【課題7】nobody所有ファイルの配置 (ファイル検索の前提条件)
	// -------------------------------------------------------------
	fmt.Println("[3/4] 課題7用の「nobody」所有のダミーファイルをシステム内に配置中...")

	nobodyUser, err := user.Lookup("nobody")
	if err != nil {
		fmt.Printf("エラー: nobody ユーザーが見つかりません: %v\n", err)
		os.Exit(1)
	}
	uid, _ := strconv.Atoi(nobodyUser.Uid)
	gid, _ := strconv.Atoi(nobodyUser.Gid)

	os.MkdirAll("/opt/nobody_data", 0755)
	dummyFiles := []string{
		"/opt/nobody_data/secure_report.dat",
		"/var/tmp/system_process_nobody.log",
		"/tmp/session_cache_nobody.txt",
	}
	for _, file := range dummyFiles {
		if !checkNobodyOwner(file) {
			f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Printf("エラー: ダミーファイルの作成に失敗しました (%s): %v\n", file, err)
				os.Exit(1)
			}
			f.Close()
			if err := os.Chown(file, uid, gid); err != nil {
				fmt.Printf("エラー: 所有者の変更に失敗しました (%s): %v\n", file, err)
				os.Exit(1)
			}
		}
	}
	fmt.Println("✔ nobody 所有のファイルを配置または確認しました。")

	// -------------------------------------------------------------
	// 5. 【課題14】シェルスクリプトテスト用ディレクトリの作成
	// -------------------------------------------------------------
	fmt.Println("[4/4] 課題14の動作検証用テストディレクトリを配置中...")

	testDir := "/var/tmp/script_test_env"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.MkdirAll(testDir+"/sub_directory_to_ignore", 0755)
		filesToCreate := []string{
			testDir + "/sample_file1.txt",
			testDir + "/sample_file2.log",
			testDir + "/sample_file3.conf",
			testDir + "/sub_directory_to_ignore/should_not_be_listed.txt",
		}
		for _, fPath := range filesToCreate {
			f, err := os.Create(fPath)
			if err != nil {
				fmt.Printf("エラー: テストファイルの作成に失敗しました (%s): %v\n", fPath, err)
				os.Exit(1)
			}
			f.Close()
		}
		fmt.Printf("✔ スクリプトテスト環境を %s に配置しました。\n", testDir)
	} else {
		fmt.Println("✔ スクリプトテスト環境は既に存在します。スキップします。")
	}

	fmt.Println("=====================================================")
	fmt.Println("🎉 環境セットアップが正常に完了しました！")
	fmt.Println("これで Canvas 内の全14問に挑戦する準備が整いました。")
	fmt.Println("=====================================================")
}
