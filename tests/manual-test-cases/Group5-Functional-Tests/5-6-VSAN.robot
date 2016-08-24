*** Settings ***
Documentation  Test 5-6 - VSAN
Resource  ../../resources/Util.robot
Suite Teardown  Run Keyword And Ignore Error  Kill Nimbus Server  %{NIMBUS_USER}  %{NIMBUS_PASSWORD}  *

*** Test Cases ***
Test
    ${out}=  Deploy Nimbus Testbed  %{NIMBUS_USER}  %{NIMBUS_PASSWORD}  --vcvaBuild 3634791 --esxBuild 3620759 --testbedName vcqa-standard-iscsi-fullInstall-vcva --runName VIC-VSAN-Test
    Should Contain  ${out}  Testbed VIC-VSAN-Test available ...
    ${lines}=  Split To Lines  ${out}
    :FOR  ${line}  IN  @{lines}
    \   ${status}=  Run Keyword And Return Status  Should Match Regexp  ${line}  .esx.0' is up. IP:
    \   ${ip}=  Run Keyword If  ${status}  Fetch From Right  ${line}  ${SPACE}
    \   Run Keyword If  ${status}  Set Test Variable  ${esx1}  ${ip}
    \   ${status}=  Run Keyword And Return Status  Should Match Regexp  ${line}  .esx.1' is up. IP:
    \   ${ip}=  Run Keyword If  ${status}  Fetch From Right  ${line}  ${SPACE}
    \   Run Keyword If  ${status}  Set Test Variable  ${esx2}  ${ip}
    \   ${status}=  Run Keyword And Return Status  Should Match Regexp  ${line}  .vcva-3634791' is up. IP:
    \   ${ip}=  Run Keyword If  ${status}  Fetch From Right  ${line}  ${SPACE}
    \   Run Keyword If  ${status}  Set Test Variable  ${vc}  ${ip}

    Log To Console  Set environment variables up for GOVC
    Set Environment Variable  GOVC_URL  ${vc}
    Set Environment Variable  GOVC_USERNAME  Administrator@vsphere.local
    Set Environment Variable  GOVC_PASSWORD  Admin\!23
    Set Environment Variable  GOVC_DATASTORE  sharedVmfs-0

    Log To Console  Create a distributed switch
    ${out}=  Run  govc dvs.create -dc=vcqaDC test-ds
    Should Contain  ${out}  OK

    Log To Console  Create three new distributed switch port groups for management and vm network traffic
    ${out}=  Run  govc dvs.portgroup.add -nports 12 -dc=vcqaDC -dvs=test-ds management
    Should Contain  ${out}  OK
    ${out}=  Run  govc dvs.portgroup.add -nports 12 -dc=vcqaDC -dvs=test-ds vm-network
    Should Contain  ${out}  OK
    ${out}=  Run  govc dvs.portgroup.add -nports 12 -dc=vcqaDC -dvs=test-ds bridge
    Should Contain  ${out}  OK
    Set Environment Variable  BRIDGE_NETWORK  bridge
    Set Environment Variable  EXTERNAL_NETWORK  vm-network

    Log To Console  Add the ESXi hosts to the portgroups
    ${hosts}=  Run  govc ls host
    ${hosts}=  Split To Lines  ${hosts}
    :FOR  ${host}  IN  @{hosts}
    \   ${out}=  Run  govc dvs.add -dvs=test-ds -pnic=vmnic1 ${host}
    \   Should Contain  ${out}  OK
    
    ${out}=  Run  govc cluster.change -drs-enabled /vcqaDC/host/cls

    Set Environment Variable  TEST_URL_ARRAY  ${vc}
    Set Environment Variable  TEST_USERNAME  Administrator@vsphere.local
    Set Environment Variable  TEST_PASSWORD  Admin\!23
    Set Environment Variable  TEST_DATASTORE  sharedVmfs-0
    Set Environment Variable  TEST_RESOURCE  cls
    Install VIC Appliance To Test Server  ${false}  default
    
    Run Regression Tests
    
    #nimbus-testbeddeploy --vcvaBuild 3634791 --esxBuild 3620759 --testbedName test-vpx-4esx-virtual-fullInstall-vcva-8gbmem --runName VIC-VSAN-Test