#!/bin/bash
# =====================================================================
#  RHCSA EX200 v10 (RHEL 10) 模擬試験用 環境セットアップ・初期化スクリプト
#  ※必ず root ユーザー、または sudo を使用して実行してください。
# =====================================================================

# 実行ユーザーのチェック
if [ "$EUID" -ne 0 ]; then
    echo "エラー: このスクリプトは root 権限で実行する必要があります。"
    exit 1
fi

echo "====================================================="
echo "   RHCSA EX200 v10 模擬試験環境の構築を開始します"
echo "====================================================="

# -------------------------------------------------------------
# 1. クリーンアップが必要かどうかの事前チェック
# -------------------------------------------------------------
echo "システム上の古い模擬試験環境を検出しています..."

need_cleanup=false

# マウントのチェック
if mountpoint -q /mnt/store_data 2>/dev/null; then need_cleanup=true; fi

# LVM (物理ボリューム、ボリュームグループ、論理ボリューム) のチェック
if lvs vg_store &>/dev/null || vgs vg_store &>/dev/null || pvs /dev/vdb &>/dev/null || pvs /dev/sdb &>/dev/null; then 
    need_cleanup=true 
fi

# ループバックイメージおよびシンボリックリンクのチェック
if [ -f /var/lib/mock_extra_disk.img ] || [ -L /dev/vdb ] || [ -e /dev/vdb ]; then 
    need_cleanup=true 
fi

# ユーザー・グループのチェック
if id adminuser &>/dev/null || id devuser &>/dev/null; then need_cleanup=true; fi
if getent group sysops &>/dev/null || getent group devs &>/dev/null; then need_cleanup=true; fi

# 設定ファイルのチェック
if [ -f /etc/sudoers.d/sysops ]; then need_cleanup=true; fi
if [ -d /common/shared_ops ] || [ -d /mnt/store_data ] || [ -d /root/found_files ]; then need_cleanup=true; fi
if [ -f /root/etc_backup.tar.gz ] || [ -f /usr/local/bin/dir_check.sh ]; then need_cleanup=true; fi

# systemd タイマーおよびサービスのチェック
if [ -f /etc/systemd/system/sys-cleanup.service ] || [ -f /etc/systemd/system/sys-cleanup.timer ]; then 
    need_cleanup=true 
fi
if systemctl is-active sys-cleanup.timer &>/dev/null; then need_cleanup=true; fi

# fstab 内の記述チェック
if [ -f /etc/fstab ]; then
    if grep -qE "vg_store|lv_store|swap" /etc/fstab; then need_cleanup=true; fi
fi

# ホスト名のチェック
CURRENT_HOSTNAME=$(hostname)
if [ "$CURRENT_HOSTNAME" != "localhost.localdomain" ] && [ "$CURRENT_HOSTNAME" != "localhost" ]; then 
    need_cleanup=true 
fi


# -------------------------------------------------------------
# 2. クリーンアップの実行またはスキップ
# -------------------------------------------------------------
if [ "$need_cleanup" = "true" ]; then
    echo "[1/4] 既存の模擬試験設定・回答のクリーンアップを実行中..."

    # ディレクトリマウントの解除とLVMの削除
    umount /mnt/store_data &>/dev/null || true
    lvremove -y /dev/vg_store/lv_store &>/dev/null || true
    vgremove -y vg_store &>/dev/null || true
    pvremove -y /dev/vdb &>/dev/null || true
    pvremove -y /dev/sdb &>/dev/null || true

    # ループバックディスクのクリーンアップ
    if [ -f /var/lib/mock_extra_disk.img ]; then
        LOOP_DEVS=$(losetup -j /var/lib/mock_extra_disk.img | cut -d: -f1)
        for dev in $LOOP_DEVS; do
            losetup -d "$dev" &>/dev/null || true
        done
        rm -f /var/lib/mock_extra_disk.img
    fi
    rm -f /dev/vdb

    # ユーザー・グループの削除
    userdel -r adminuser &>/dev/null || true
    userdel -r devuser &>/dev/null || true
    groupdel sysops &>/dev/null || true
    groupdel devs &>/dev/null || true
    rm -f /etc/sudoers.d/sysops

    # 各種作成ファイルの削除
    rm -rf /common/shared_ops
    rm -rf /mnt/store_data
    rm -rf /root/found_files
    rm -f /root/etc_backup.tar.gz
    rm -f /usr/local/bin/dir_check.sh

    # systemd タイマーの停止・削除
    systemctl stop sys-cleanup.timer &>/dev/null || true
    systemctl disable sys-cleanup.timer &>/dev/null || true
    rm -f /etc/systemd/system/sys-cleanup.service
    rm -f /etc/systemd/system/sys-cleanup.timer
    systemctl daemon-reload

    # fstab から試験で追加された設定行を削除
    if [ -f /etc/fstab ]; then
        sed -i '/vg_store-lv_store/d' /etc/fstab || true
        sed -i '/lv_store/d' /etc/fstab || true
        sed -i '/swap/d' /etc/fstab || true
        swapoff -a &>/dev/null || true
        rm -f /swapfile &>/dev/null || true
        rm -f /mock_swap &>/dev/null || true
        swapon -a &>/dev/null || true
    fi

    # ホスト名の初期化
    hostnamectl set-hostname localhost.localdomain

    echo "✔ クリーンアップが完了しました。"
else
    echo "[1/4] クリーンアップの対象となる不要な設定やファイルは検出されませんでした。スキップします。"
fi


# -------------------------------------------------------------
# 3. 追加ディスクのシミュレーション (5GB)
# -------------------------------------------------------------
echo "[2/4] 追加ディスク (/dev/vdb 5GB) のシミュレーション設定はスキップされました（LVM設定ロジック削除）。"


# -------------------------------------------------------------
# 4. 【課題7】nobody所有ファイルの配置 (ファイル検索の前提条件)
# -------------------------------------------------------------
echo "[3/4] 課題7用の「nobody」所有のダミーファイルをシステム内に配置中..."

# ファイルが既に存在してnobody所有であるかチェックし、なければ配置
mkdir -p /opt/nobody_data
for file in "/opt/nobody_data/secure_report.dat" "/var/tmp/system_process_nobody.log" "/tmp/session_cache_nobody.txt"; do
    if [ ! -f "$file" ] || [ "$(stat -c '%U' "$file" 2>/dev/null)" != "nobody" ]; then
        touch "$file"
        chown nobody:nobody "$file"
    fi
done

echo "✔ nobody 所有のファイルを配置または確認しました。"


# -------------------------------------------------------------
# 5. 【課題14】シェルスクリプトテスト用ディレクトリの作成
# -------------------------------------------------------------
echo "[4/4] 課題14の動作検証用テストディレクトリを配置中..."

TEST_DIR="/var/tmp/script_test_env"
if [ ! -d "$TEST_DIR" ]; then
    mkdir -p "$TEST_DIR/sub_directory_to_ignore"
    touch "$TEST_DIR/sample_file1.txt"
    touch "$TEST_DIR/sample_file2.log"
    touch "$TEST_DIR/sample_file3.conf"
    touch "$TEST_DIR/sub_directory_to_ignore/should_not_be_listed.txt"
    echo "✔ スクリプトテスト環境を $TEST_DIR に配置しました。"
else
    echo "✔ スクリプトテスト環境は既に存在します。スキップします。"
fi

echo "====================================================="
echo "🎉 環境セットアップが正常に完了しました！"
echo "これで Canvas 内の全14問に挑戦する準備が整いました。"
echo "====================================================="
