*** Settings ***
Documentation  Test 5-7 - NSX
Resource  ../../resources/Util.robot

*** Test Cases ***
Test
    ${out}=  Deploy Nimbus Testbed  --testbedName test-vpx-4esx-virtual-fullInstall-vcva-8gbmem-nsx1m1c