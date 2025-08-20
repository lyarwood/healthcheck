# healthcheck

A basic command line tool to help summarise CI failures as reported by the [kubevirt/ci-health](https://github.com/kubevirt/ci-health) project.

## Lane Analysis

Analyze recent job runs for a specific lane with various output options:

```shell
# Count test failures across recent runs
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 5 -c
2	VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15447/pull-kubevirt-unit-test-arm64/1958193812496977920


1	Migration watcher Migration backoff should not be applied if it is not an evacuation with workload update annotation

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15388/pull-kubevirt-unit-test-arm64/1958193968416034816
```

```shell
# Show only test names
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 3 -n
VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts
Migration watcher Migration backoff should not be applied if it is not an evacuation with workload update annotation
```

```shell
# Show only failed job URLs
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 3 -u
https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920
https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15388/pull-kubevirt-unit-test-arm64/1958193968416034816
```

```shell
# Filter by time period with automatic pagination (finds ALL results within time period)
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 1h -c
1	VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920

# Analyze longer time periods - automatically paginates to find all results
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 2d --summary
Lane Summary: pull-kubevirt-unit-test-arm64
===========================================

Time Range:
  First Run:  2025-08-18 19:03:13 UTC
  Last Run:   2025-08-20 16:22:12 UTC
  Duration:   1.9 days

Test Run Statistics:
  Total Runs:     92
  Successful:     62
  Failed:         15
  Unknown:        15
  Failure Rate:   16.3%
```

```shell
# Get a concise summary with failure patterns and statistics
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 5 --summary
Lane Summary: pull-kubevirt-unit-test-arm64
===========================================

Time Range:
  First Run:  2025-08-20 14:32:15 UTC
  Last Run:   2025-08-20 16:22:12 UTC
  Duration:   1.8 hours

Test Run Statistics:
  Total Runs:     5
  Successful:     0
  Failed:         4
  Unknown:        1
  Failure Rate:   80.0%

Test Failure Statistics:
  Total Failures: 4
  Unique Tests:   2

Failure Categories:
  migration : 4 (100.0%)

Most Frequent Failures:
  1. [migration] VirtualMachineInstance migration target DomainNotifyServe... (3 failures, 75.0%)
  2. [migration] Migration watcher Migration backoff should not be applied... (1 failures, 25.0%)

Pattern Analysis:
  🟡 Moderate failure rate - monitor trends
  🎯 Single dominant failure pattern (migration)
```

## Open a tab for each sig-compute job failure

```shell
$ healthcheck merge -u -j compute | sort | uniq | xargs google-chrome
```

## List only failing test names and count with external tools

```shell
$ healthcheck merge -n | sort | uniq -c | sort -rn
      3 [sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates
      3 [sig-compute] Infrastructure cluster profiler for pprof data aggregation when ClusterProfiler configuration is enabled it should allow subresource access
      2 [virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times
      2 [sig-storage] Hotplug [storage-req] VMI migration should allow live migration with attached hotplug volumes containerDisk VMI
      1 [sig-storage] Volumes update with migration Update volumes with the migration updateVolumesStrategy should be able to recover from an interrupted volume migration when the copy of the destination volumes was successful
      1 [sig-storage] Volumes update with migration Hotplug volumes should be able to volume migrate a VM with a datavolume and an hotplugged datavolume migrating from block to filesystem
      1 [sig-storage] Storage Starting a VirtualMachineInstance [storage-req][rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine block volume PVC [test_id:3139]should be successfully started
      1 [sig-storage] Hotplug [storage-req] VMI migration should allow live migration with attached hotplug volumes persistent disk VMI
      1 [sig-operator]Operator [rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]infrastructure management [test_id:3151]should be able to update kubevirt install when operator updates if no custom image tag is set
      1 [sig-operator]Operator [rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]infrastructure management [test_id:3150]should be able to update kubevirt install with custom image tag
      1 [sig-operator]Operator  Deployment of common-instancetypes external CA should properly manage adding entries to the configmap
      1 [sig-network]  VirtualMachineInstance with passt network binding plugin migration connectivity should be preserved [IPv4]
      1 [sig-network] [rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Networking Multiple virtual machines connectivity using bridge binding interface with a test outbound VMI should be able to reach [test_id:1539]the Inbound VirtualMachineInstance with default (implicit) binding
      1 [sig-network] bridge nic-hotplug a running VM is able to hotplug multiple network interfaces Migration based
      1 [sig-compute]VirtualMachinePool pool should scale to five, to six and then to zero replicas
      1 [sig-compute] [rfe_id:1177][crit:medium] VirtualMachine with paused vmi [test_id:3229]should gracefully handle being started again
      1 [sig-compute]Memory Hotplug A VM with memory liveUpdate enabled should detect a failed memory hotplug
      1 [sig-compute] Infrastructure [rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates [test_id:4099] should be rotated when a new CA is created
      1 [sig-compute] Infrastructure [rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints should include the storage metrics for a running VM [test_id:6230] I/O read operations metric by using IPv6
      1 [sig-compute] Infrastructure [rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints should include metrics for a running VM [test_id:4143] network metrics by IPv4
      1 [sig-compute]Configurations VirtualMachineInstance definition [rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with guestAgent with cluster config changes [test_id:6958]VMI condition should not signal unsupported agent presence for optional commands
      1 [sig-compute]Configurations [rfe_id:904][crit:medium][vendor:cnv-qe@redhat.com][level:component]with driver cache and io settings and PVC [test_id:1681]should set appropriate cache modes
      1 [rfe_id:899][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Config With a DownwardAPI defined [test_id:790]Should be the namespace and token the same for a pod and vmi
      1 [rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]ContainerDisk [rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting and stopping the same VirtualMachine with ephemeral registry disk [test_id:1463] should success multiple times
      1 [rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]VMIlifecycle Softreboot a VirtualMachineInstance soft reboot vmi with ACPI feature enabled should succeed
      1 [rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]VMIlifecycle [rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance with user-data without k8s secret [test_id:1630]should log warning and proceed once the secret is there
      1 [rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given [test_id:1527]should not update the VirtualMachineInstance spec if Running
      1 [rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given [test_id:1526]should start and stop VirtualMachineInstance multiple times
      1 [rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given [test_id:1523]should recreate VirtualMachineInstance if it gets deleted
      1 [rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given should not update the vmi generation annotation when the template changes
      1 [rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given should not remove a succeeded VMI [test_id:2190] with RunStrategyManual
      1 pull-kubevirt-e2e-k8s-1.31-sig-storage (no junit file to parse)
      1 pull-kubevirt-e2e-k8s-1.31-sig-network (no junit file to parse)
```

## Count and list test failures with job URLs

```shell
$ healthcheck merge -c -j compute
3 [sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15182/pull-kubevirt-e2e-k8s-1.31-sig-compute/1945105449749581824

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15122/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943094557549793280


2 [virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.32-sig-compute/1943363976574275584

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15099/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943378428849819648


1 [sig-compute]VirtualMachinePool pool should scale to five, to six and then to zero replicas

  https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.31-sig-compute/1943363975836078080

[..]
```

## Count and list test failures with any additional failure context

```shell
$ healthcheck merge -c -f -j compute
3 [sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates

 {{ failure}  Failure tests/pool_test.go:701
Expected
    <int>: 3
to equal
    <int>: 4
tests/pool_test.go:760}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

 {{ failure}  Failure tests/pool_test.go:701
Timed out after 90.001s.
Expected
    <int32>: 2
to equal
    <int32>: 4
tests/pool_test.go:108}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15182/pull-kubevirt-e2e-k8s-1.31-sig-compute/1945105449749581824

 {{ failure}  Failure tests/pool_test.go:701
Timed out after 90.001s.
Expected
    <int32>: 3
to equal
    <int32>: 4
tests/pool_test.go:108}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15122/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943094557549793280


2 [virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times

 {{ failure}  Failure tests/virtctl/usbredir.go:74
Timed out after 90.001s.
Timed out waiting for VMI testvmi-mv6sl to enter [Running] phase(s)
Expected
    <v1.VirtualMachineInstancePhase>: Scheduled
to be an element of
    <[]v1.VirtualMachineInstancePhase | len:1, cap:1>: ["Running"]
tests/libwait/wait.go:76}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.32-sig-compute/1943363976574275584

 {{ failure}  Failure tests/virtctl/usbredir.go:74
Timed out after 90.001s.
Timed out waiting for VMI testvmi-lgflv to enter [Running] phase(s)
Expected
    <v1.VirtualMachineInstancePhase>: Scheduled
to be an element of
    <[]v1.VirtualMachineInstancePhase | len:1, cap:1>: ["Running"]
tests/libwait/wait.go:76}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15099/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943378428849819648


1 [sig-compute]VirtualMachinePool pool should scale to five, to six and then to zero replicas

 {{ failure}  Failure tests/pool_test.go:174
Timed out after 90.000s.
Expected
    <int32>: 5
to equal
    <int32>: 6
tests/pool_test.go:108}
 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.31-sig-compute/1943363975836078080

```
