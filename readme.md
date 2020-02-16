# cni-terway

此插件借助kubernetescni插件中的`bridge`和`dhcp`, 实现了类似于虚拟机的桥接模式网络. 

在此网络模型下创建的Pod可以获得宿主机网络的IP, Pod与宿主机网络中的主机可以直接通信.

此插件的工作流程十分简单, 只有如下两步:

1. 创建bridge接口, 连接物理网卡与bridge, 然后修改路由(从物理网卡改为通过bridge设备);
2. 启动dhcp服务守护进程(其实`dhcp`插件在工程代码中就给出了`.service`服务脚本, 本来建议使用`systemctl`将其作为服务启动的, 但是我更希望在Pod中集成这样的功能, 减少多余的操作).

## 网络拓扑变化

### 1.

```
+-----------------------------------------------------------+
|  +-----------------------------------------------------+  |
|  |                 Newwork Protocol Stack              |  |
|  +------↑-----------------------↑-------------------↑--+  |
|         |                       |                   |     |
|.........|.......................|...................|.....|
|         |                       |                   |     |
|         |         +--------+    |    +--------+     |     |
|         |         |  pod1  |    |    |  pod2  |     |     |
|         |         +----↑---+    |    +---↑----+     |     |
|         |              └─────┐  |  ┌─────┘          |     |
| +-------↓-------+         +--↓--↓--↓--+          ***↓**** |
| |      eth0     |         |   mybr0   |          * dhcp * |
| |192.168.0.10/24|         +-----------+          ***↓**** |
| +-------↑-------+                                         |
+---------|-------------------------------------------------+
          ↓
```

### 2. 

```
+-----------------------------------------------------------+
|  +-----------------------------------------------------+  |
|  |                 Newwork Protocol Stack              |  |
|  +------------------------------↑-------------------↑--+  |
|                                 |                   |     |
|.................................|...................|.....|
|                                 |                   |     |
|                   +--------+    |    +--------+     |     |
|                   |  pod1  |    |    |  pod2  |     |     |
|                   +----↑---+    |    +---↑----+     |     |
|                        └─────┐  |  ┌─────┘          |     |
|                         +----↓--↓--↓----+           |     |
|   +-----------+         |     mybr0     |        ***↓**** |
|   |   eth0    |<------->|192.168.0.10/24|        * dhcp * |
|   +-----↑-----+         +---------------+        ******** |
+---------|-------------------------------------------------+
          ↓
```

在此网络模式中, 物理网卡`eth0`退化为一条网线, 由`mybr0`接管其原有IP, 对外连接物理网络, 各节点之间相当于通过交换机直接相连.

## 注意

由于此插件使用桥接网络, 所以会使`kubeadm --config`配置文件中的`networking.podSubnet`字段(或是`--pod-network-cidr`选项)失效.

## 开发流程

编译

```
$ go build -o terway main.go
$ go build -o cni-terway ./cni/main.go
```

