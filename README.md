# healthcheck

A basic command line tool to help summarise CI failures as reported by the [kubevirt/ci-health](https://github.com/kubevirt/ci-health) project.

```shell

# Open a tab for each sig-compute$ failure 
$ healthcheck -u -j "sig-compute$" | xargs google-chrome

# List a sorted list of job failures
$ healthcheck -c -j "sig-compute$"
3 [sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15182/pull-kubevirt-e2e-k8s-1.31-sig-compute/1945105449749581824

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15122/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943094557549793280


2 [virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.32-sig-compute/1943363976574275584

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15099/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943378428849819648


1 [sig-compute]VirtualMachinePool pool should scale to five, to six and then to zero replicas

 https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.31-sig-compute/1943363975836078080

# List a sorted list of job failures and also print out any failure context
$ healthcheck -c -f -j "sig-compute$" 
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
