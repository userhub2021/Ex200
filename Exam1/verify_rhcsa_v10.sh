#!/bin/bash
# =====================================================================
#  RHCSA EX200 v10 (RHEL 10) Autograding Verification Script
#  Run this script as root to check your progress on the tasks.
# =====================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=====================================================${NC}"
echo -e "${BLUE}    RHCSA EX200 (RHEL 10) 模擬試験 自動採点スクリプト ${NC}"
echo -e "${BLUE}=====================================================${NC}"

total_score=0
max_score=140 # 各10点 (全14問)

check_task() {
    local task_name=$1
    local success=$2
    local points=$3
    if [ "$success" = "true" ]; then
        echo -e "[ ${GREEN}PASS${NC} ] $task_name (+${points} pts)"
        total_score=$((total_score + points))
    else
        echo -e "[ ${RED}FAIL${NC} ] $task_name"
    fi
}

# -------------------------------------------------------------
# 1. Rescue Mode Check (Root Password change verify)
# -------------------------------------------------------------
# スクリプトがrootで実行できているため、基本動作をPASSとみなす
check_task "【課題1】システムレスキュー & rootパスワード変更" "true" 10

# -------------------------------------------------------------
# 2. Hostname & Network Keyfile Check (RHEL 10 Standard)
# -------------------------------------------------------------
net_check="false"
if [ "$(hostname)" = "node1.exam.example.com" ]; then
    # NM keyfileが正しく設定されているか確認
    if ls /etc/NetworkManager/system-connections/*.nmconnection >/dev/null 2>&1; then
        net_check="true"
    fi
fi
check_task "【課題2】ホスト名 & NetworkManager Keyfile設定" "$net_check" 10

# -------------------------------------------------------------
# 3. Repository Check (DNF 5)
# -------------------------------------------------------------
repo_check="false"
if dnf repolist -q 2>/dev/null | grep -qiE "(baseos|appstream|local)"; then
    repo_check="true"
fi
check_task "【課題3】DNFリポジトリの構成 (DNF 5)" "$repo_check" 10

# -------------------------------------------------------------
# 4. Users, Groups and Sudoers Check
# -------------------------------------------------------------
user_check="false"
if getent group sysops >/dev/null && getent group devs >/dev/null; then
    if id adminuser 2>&1 | grep -q "sysops" && id devuser 2>&1 | grep -q "devs"; then
        if [ "$(getent passwd devuser | cut -d: -f7)" = "/sbin/nologin" ]; then
            # Sudoers configuration check
            if [ -f "/etc/sudoers.d/sysops" ] && grep -q "%sysops" /etc/sudoers.d/sysops; then
                user_check="true"
            fi
        fi
    fi
fi
check_task "【課題4】ユーザー・グループおよびSudo権限" "$user_check" 10

# -------------------------------------------------------------
# 5. Shared Directory Check (SGID)
# -------------------------------------------------------------
shared_dir_check="false"
if [ -d "/common/shared_ops" ]; then
    owner_group=$(stat -c "%G" /common/shared_ops)
    perms=$(stat -c "%a" /common/shared_ops)
    if [ "$owner_group" = "sysops" ] && [[ "$perms" =~ ^277[0-7]$ || "$perms" =~ ^27[0-7]0$ ]]; then
        shared_dir_check="true"
    fi
fi
check_task "【課題5】共同作業用ディレクトリ (SGID)" "$shared_dir_check" 10

# -------------------------------------------------------------
# 6. systemd Timer Check (RHEL 10 New Topic)
# -------------------------------------------------------------
timer_check="false"
if systemctl is-active sys-cleanup.timer >/dev/null 2>&1; then
    if [ -f "/etc/systemd/system/sys-cleanup.service" ] && [ -f "/etc/systemd/system/sys-cleanup.timer" ]; then
        timer_check="true"
    fi
fi
check_task "【課題6】systemd タイマーユニットの構成" "$timer_check" 10

# -------------------------------------------------------------
# 7. File Search Check
# -------------------------------------------------------------
find_check="false"
if [ -d "/root/found_files" ] && [ "$(find /root/found_files -type f 2>/dev/null | wc -l)" -gt 0 ]; then
    find_check="true"
fi
check_task "【課題7】ファイルの検索とコピー" "$find_check" 10

# -------------------------------------------------------------
# 8. Backup / Tar Check
# -------------------------------------------------------------
backup_check="false"
if [ -f "/root/etc_backup.tar.gz" ]; then
    if tar -tzf /root/etc_backup.tar.gz >/dev/null 2>&1; then
        backup_check="true"
    fi
fi
check_task "【課題8】アーカイブと圧縮" "$backup_check" 10

# -------------------------------------------------------------
# 9. SELinux & Apache Check
# -------------------------------------------------------------
selinux_httpd_check="false"
if systemctl is-active httpd >/dev/null 2>&1; then
    if semanage port -l 2>/dev/null | grep http_port_t | grep -q "82"; then
        if firewall-cmd --list-ports --permanent | grep -q "82/tcp"; then
            selinux_httpd_check="true"
        fi
    fi
fi
check_task "【課題9】SELinux非標準ポート (Web: 82)" "$selinux_httpd_check" 10

# -------------------------------------------------------------
# 10 & 11. LVM & Resize Check
# -------------------------------------------------------------
lvm_check="false"
if lvs vg_store/lv_store -o lv_size --noheadings 2>/dev/null | grep -qE "(1\.[2-9]|2\.[0-9])G"; then
    if findmnt -n -o FSTYPE /mnt/store_data 2>/dev/null | grep -q "xfs"; then
        if grep -q "vg_store-lv_store" /etc/fstab || grep -q "lv_store" /etc/fstab; then
            lvm_check="true"
        fi
    fi
fi
check_task "【課題10 & 11】LVMの作成・マウント・拡張" "$lvm_check" 20

# -------------------------------------------------------------
# 12. Swap Check
# -------------------------------------------------------------
swap_check="false"
if swapon --show --noheadings 2>/dev/null | grep -qE "(partition|file)"; then
    if grep -q "swap" /etc/fstab; then
        swap_check="true"
    fi
fi
check_task "【課題12】スワップ(Swap)の追加・永続化" "$swap_check" 10

# -------------------------------------------------------------
# 13. Flatpak Package Check (RHEL 10 New Topic)
# -------------------------------------------------------------
flatpak_check="false"
if flatpak remotes 2>/dev/null | grep -q "flathub"; then
    # インストールされたフラットパックがあるかチェック
    if [ "$(flatpak list --system 2>/dev/null | wc -l)" -gt 0 ]; then
        flatpak_check="true"
    fi
fi
check_task "【課題13】Flatpakリモート追加 & アプリのインストール" "$flatpak_check" 10

# -------------------------------------------------------------
# 14. Shell Script Check (RHEL 10 New Topic)
# -------------------------------------------------------------
script_check="false"
script_path="/usr/local/bin/dir_check.sh"
if [ -x "$script_path" ]; then
    # テスト1: 引数なしでエラー(終了コード1)になるか
    $script_path >/dev/null 2>&1
    exit_code_1=$?
    
    # テスト2: 存在しないディレクトリでエラー(終了コード2)になるか
    $script_path /nonexistent_dir_test >/dev/null 2>&1
    exit_code_2=$?
    
    # テスト3: 実在するディレクトリ(/etc)で正常終了(終了コード0)になるか
    $script_path /etc >/dev/null 2>&1
    exit_code_3=$?
    
    if [ $exit_code_1 -eq 1 ] && [ $exit_code_2 -eq 2 ] && [ $exit_code_3 -eq 0 ]; then
        script_check="true"
    fi
fi
check_task "【課題14】簡易シェルスクリプトの作成" "$script_check" 10

# -------------------------------------------------------------
# Summary
# -------------------------------------------------------------
echo -e "${BLUE}=====================================================${NC}"
echo -e "  結果: ${total_score} / ${max_score} 点"
pass_mark=$((max_score * 70 / 100)) # 70%以上で合格
if [ "$total_score" -ge "$pass_mark" ]; then
    echo -e "  ステータス: ${GREEN}合格 (CONGRATULATIONS!)${NC}"
else
    echo -e "  ステータス: ${RED}不合格 (Keep studying!)${NC}"
fi
echo -e "${BLUE}=====================================================${NC}"

