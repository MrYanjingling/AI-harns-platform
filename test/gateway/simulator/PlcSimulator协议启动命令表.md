# PlcSimulator（ZL.PlcSimulator）协议启动命令表

> 工具：PlcSimulator.Cli（`plcsim`，dotnet tool）。统一单启模板：
>
> ```bash
> plcsim --protocols <协议标识> --auto-start
> ```

## 安装与环境

```bash
# 安装（需先装 .NET 8 SDK）
dotnet tool install --global PlcSimulator.Cli

# 升级
dotnet tool update --global PlcSimulator.Cli
```

macOS（zsh）安装后若提示工具目录不在 PATH：

```zsh
cat << \EOF >> ~/.zprofile
export PATH="$PATH:/Users/apple/.dotnet/tools"
EOF
source ~/.zprofile
```

常用参数：

| 参数 | 含义 |
|---|---|
| `--auto-start` | 启动即监听，无需进交互界面再 `start` |
| `--headless` | 无头模式，无交互，适合 docker / 后台 |
| `--stay-alive` | 常驻模式（须配 `--headless`），由信号触发优雅退出，适合 nohup / systemd |
| `--protocols <列表>` | 逗号分隔，只启用指定协议；不指定则启用全部 |
| `--ports <配置>` | 协议端口映射，格式 `<协议>=<端口>[,<协议>=<端口>]...`，用于错开冲突端口 |
| `--serial <协议>=<设备>` | 串口协议用哪个串口；`auto` 自动建虚拟串口对，客户端连 external |

---

## TCP 协议

| 协议标识 | 默认端口 | 单条启动命令 | 备注 |
|---|---|---|---|
| `fins-tcp` | 9600 | `plcsim --protocols fins-tcp --auto-start` | 欧姆龙 FINS/TCP |
| `omron-host-link` | 9600 | `plcsim --protocols omron-host-link --auto-start` | Host Link TCP 仿真；与 fins-tcp 同启需错开 |
| `omron-njnx` | 44818 | `plcsim --protocols omron-njnx --auto-start` | NJ/NX EtherNet/IP/CIP |
| `allen-bradley` | 44818 | `plcsim --protocols allen-bradley --auto-start` | |
| `ethernet-ip` | 44818 | `plcsim --protocols ethernet-ip --auto-start` | |
| `allen-bradley-pccc` | 11111 | `plcsim --protocols allen-bradley-pccc --auto-start` | |
| `beckhoff-ads` | 48898 | `plcsim --protocols beckhoff-ads --auto-start` | |
| `cimon` | 9300 | `plcsim --protocols cimon --auto-start` | |
| `cjt188` | 37121 | `plcsim --protocols cjt188 --auto-start` | |
| `dlt645` | 2010 | `plcsim --protocols dlt645 --auto-start` | |
| `fanuc` | 8193 | `plcsim --protocols fanuc --auto-start` | |
| `fatek` | 500 | `plcsim --protocols fatek --auto-start` | |
| `fuji-command-setting` | 9100 | `plcsim --protocols fuji-command-setting --auto-start` | |
| `fuji-spb` | 9110 | `plcsim --protocols fuji-spb --auto-start` | |
| `fuji-sph` | 9120 | `plcsim --protocols fuji-sph --auto-start` | |
| `ge-srtp` | 18245 | `plcsim --protocols ge-srtp --auto-start` | |
| `hart-ip` | 5094 | `plcsim --protocols hart-ip --auto-start` | |
| `iec104` | 2404 | `plcsim --protocols iec104 --auto-start` | |
| `iec61850` | 102 | `plcsim --protocols iec61850 --auto-start` | |
| `inovance-easynet` | 5000 | `plcsim --protocols inovance-easynet --auto-start` | |
| `keyence-mc-ascii` | 5006 | `plcsim --protocols keyence-mc-ascii --auto-start` | |
| `keyence-nano` | 8501 | `plcsim --protocols keyence-nano --auto-start` | |
| `lscnet` | 2004 | `plcsim --protocols lscnet --auto-start` | |
| `lsis` | 2004 | `plcsim --protocols lsis --auto-start` | |
| `melsec-a3c` | 5000 | `plcsim --protocols melsec-a3c --auto-start` | |
| `melsec-a4c` | 5000 | `plcsim --protocols melsec-a4c --auto-start` | |
| `melsec-fx-links` | 8000 | `plcsim --protocols melsec-fx-links --auto-start` | |
| `melsec-fx-serial` | 5010 | `plcsim --protocols melsec-fx-serial --auto-start` | |
| `melsec-mc` | 5000 | `plcsim --protocols melsec-mc --auto-start` | |
| `melsec-mc-ascii` | 5002 | `plcsim --protocols melsec-mc-ascii --auto-start` | |
| `melsec-mc-ascii-udp` | 5002 | `plcsim --protocols melsec-mc-ascii-udp --auto-start` | |
| `memobus-tcp` | 502 | `plcsim --protocols memobus-tcp --auto-start` | |
| `modbus-ascii-over-tcp` | 1503 | `plcsim --protocols modbus-ascii-over-tcp --auto-start` | |
| `modbus-rtu-over-tcp` | 1502 | `plcsim --protocols modbus-rtu-over-tcp --auto-start` | |
| `modbus-tcp` | 502 | `plcsim --protocols modbus-tcp --auto-start` | |
| `mqtt_broker` | 1883 | `plcsim --protocols mqtt_broker --auto-start` | |
| `mtconnect` | 5005 | `plcsim --protocols mtconnect --auto-start` | |
| `opcua` | 4840 | `plcsim --protocols opcua --auto-start` | |
| `opcua_pubsub` | 4841 | `plcsim --protocols opcua_pubsub --auto-start` | |
| `panasonic-mc` | 5000 | `plcsim --protocols panasonic-mc --auto-start` | |
| `panasonic-mewtocol` | 9090 | `plcsim --protocols panasonic-mewtocol --auto-start` | |
| `restapi` | 8081 | `plcsim --protocols restapi --auto-start` | |
| `siemens-fetch-write` | 1100 | `plcsim --protocols siemens-fetch-write --auto-start` | |
| `siemens-ppi` | 32502 | `plcsim --protocols siemens-ppi --auto-start` | |
| `siemens-s7` | 102 | `plcsim --protocols siemens-s7 --auto-start` | |
| `toyo-puc` | 9000 | `plcsim --protocols toyo-puc --auto-start` | |
| `vigor` | 500 | `plcsim --protocols vigor --auto-start` | |
| `yokogawa-link` | 2000 | `plcsim --protocols yokogawa-link --auto-start` | |

## UDP 协议

| 协议标识 | 默认端口 | 单条启动命令 | 备注 |
|---|---|---|---|
| `omron-fins-udp` | 9600 | `plcsim --protocols omron-fins-udp --auto-start` | 欧姆龙 FINS/UDP |
| `bacnet` | 47808 | `plcsim --protocols bacnet --auto-start` | |
| `cclink-ie` | 61450 | `plcsim --protocols cclink-ie --auto-start` | |
| `melsec-mc-udp` | 6000 | `plcsim --protocols melsec-mc-udp --auto-start` | |

## 串行协议

需加 `--serial <协议标识>=auto`（`auto` 自动建虚拟串口对，客户端连 external）：

| 协议标识 | 单条启动命令 |
|---|---|
| `dali` | `plcsim --protocols dali --serial dali=auto --auto-start` |
| `dlms` | `plcsim --protocols dlms --serial dlms=auto --auto-start` |
| `dlt698` | `plcsim --protocols dlt698 --serial dlt698=auto --auto-start` |
| `dnp3` | `plcsim --protocols dnp3 --serial dnp3=auto --auto-start` |
| `iec101` | `plcsim --protocols iec101 --serial iec101=auto --auto-start` |
| `m-bus` | `plcsim --protocols m-bus --serial m-bus=auto --auto-start` |
| `modbus-ascii` | `plcsim --protocols modbus-ascii --serial modbus-ascii=auto --auto-start` |
| `modbus-rtu` | `plcsim --protocols modbus-rtu --serial modbus-rtu=auto --auto-start` |
| `omron-host-link-cmode` | `plcsim --protocols omron-host-link-cmode --serial omron-host-link-cmode=auto --auto-start` |

## 非 TCP/UDP

| 协议标识 | 说明 |
|---|---|
| `profinet` | 原始以太网，不监听 TCP/UDP 端口；`--serial`/`--station` 对其无效 |

---

## 多协议同时启动

逗号分隔 `--protocols`，撞端口用 `--ports` 错开。

```bash
# FINS/TCP + FINS/UDP（TCP/UDP 的 9600 不冲突）
plcsim --protocols fins-tcp,omron-fins-udp --auto-start

# FINS/TCP + Host Link（默认都 9600，必须错开）
plcsim --protocols fins-tcp,omron-host-link --ports fins-tcp=9600,omron-host-link=9601 --auto-start

# 后台常驻
plcsim --protocols fins-tcp --auto-start --headless --stay-alive
```

## 端口冲突需错开的组合

| 端口 | 冲突协议 |
|---|---|
| 9600 | `fins-tcp` ↔ `omron-host-link` |
| 44818 | `allen-bradley` ↔ `ethernet-ip` ↔ `omron-njnx` |
| 5000 | `melsec-mc` ↔ `melsec-a3c` ↔ `melsec-a4c` ↔ `panasonic-mc` ↔ `inovance-easynet` |
| 502 | `modbus-tcp` ↔ `memobus-tcp` |
| 2004 | `lscnet` ↔ `lsis` |
| 500 | `fatek` ↔ `vigor` |

> 直接 `plcsim --auto-start`（不带 `--protocols`）会全量启动，撞端口会被硬拦，建议按需要挑选协议。