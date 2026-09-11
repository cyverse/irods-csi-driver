# k3s test cluster for iRODS CSI Driver

Ansible playbooks that stand up a 3-node k3s cluster for manually testing the
iRODS CSI driver. The master node is tainted (`CriticalAddonsOnly=true:NoExecute`)
so it does not run test workloads.

Ansible connects to each node over SSH via its **public IP**. k3s itself is
configured to use each node's **internal IP** for node-to-node cluster
traffic (API server, kubelet, flannel); the public IP is only added as the
node's external IP and as a TLS SAN, so `kubectl` also works from outside
the cluster's internal network.

| Role    | Public IP (SSH)       | Internal IP (cluster traffic) |
|---------|------------------------|--------------------------------|
| master  | `<master-public-ip>`   | `<master-internal-ip>`         |
| worker1 | `<worker1-public-ip>`  | `<worker1-internal-ip>`        |
| worker2 | `<worker2-public-ip>`  | `<worker2-internal-ip>`        |

See `inventory.ini` for the actual IP addresses.

Assumes Ubuntu/Debian nodes with passwordless (key-based) SSH and sudo access.

## Requirements

On the control machine:
```
pip install ansible
ansible-galaxy collection install ansible.posix community.general
```

## Setup

1. Edit `inventory.ini`:
   - Set `ansible_user` to the SSH user for your VMs.
   - Uncomment/set `ansible_ssh_private_key_file` if you don't use the default key.
   - Set each host's `internal_ip` to its internal/private network address
     (used for node-to-node cluster traffic); the inventory hostname itself
     stays the public IP Ansible uses for SSH.
2. Check connectivity:
   ```
   ansible -m ping all
   ```
3. Provision the cluster:
   ```
   ansible-playbook k3s_install.yml
   ```

This installs k3s server on the master and k3s agent on both workers, and
installs a kubeconfig at `~/.kube/config` for the SSH login user on the
master node. No kubeconfig is fetched to the control machine.

The CSI node DaemonSet respects the master node's taint, so it is placed on
workers only by default.

## Using the cluster

SSH into the master node and use `kubectl` directly - it works out of the
box for the SSH login user:

```
ssh <ansible_user>@<master-public-ip>
kubectl get nodes -o wide
```

You should see all three nodes `Ready`. The master node should show the
`CriticalAddonsOnly` taint and no CSI-driver test pods should land on it.

## Tearing down

```
ansible-playbook k3s_uninstall.yml
```

Runs k3s's own uninstall scripts on every node and removes the kubeconfig
installed on the master.

## Installing irodsfsd

```
ansible-playbook irodsfsd_install.yml
```

Installs [irodsfsd](https://github.com/cyverse/irodsfsd) (the CSI driver's
pool service) on the two worker nodes only - the master never runs
workloads, so it doesn't need it. irodsfsd depends on the
[irodsfs](https://github.com/cyverse/irodsfs) FUSE client and the `fuse`
package, so the playbook installs those first, symlinks `irodsfs` to
`/usr/local/bin/irodsfs` (the path irodsfsd's default config expects). If
irodsfsd still isn't running afterwards, it tries a restart and prints a
warning rather than failing the whole run.

To remove it:

```
ansible-playbook irodsfsd_uninstall.yml
```

irodsfsd's own installer doesn't ship an uninstall script, so this stops the
service (letting it unmount anything it has mounted first) and removes the
binary, config, systemd unit, data/log directories, and the `irodsfsd`
service user/group it created.

## Notes

- Only ports needed by k3s (6443, 10250, 8472/udp, and the NodePort range
  30000-32767) are opened via `ufw`, and only if `ufw` is already installed.
  Since these are public IPs, also check any cloud/network-level security
  group and open the same ports there if traffic is still blocked.
- Set `k3s_version` in `group_vars/all.yml` to pin a specific k3s release;
  leave it empty to install the latest stable release.
