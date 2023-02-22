/*
 * Tencent is pleased to support the open source community by making TKEStack available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

 package main

 import (
	 "context"
	 goflag "flag"
	 "io/ioutil"
	 "os"
	 "strings"
 
	 vcudaapi "tkestack.io/gpu-manager/pkg/api/runtime/vcuda"
	 "tkestack.io/gpu-manager/pkg/flags"
	 "tkestack.io/gpu-manager/pkg/logs"
	 "tkestack.io/gpu-manager/pkg/utils"
 
	 "github.com/spf13/pflag"
	 "google.golang.org/grpc"
	 "k8s.io/klog"
 )
 
 var (
	 addr, busID, podUID, contName, contID string
 )
 
 func normalize(id string) string {
	 return strings.ReplaceAll(id, "_", "-")
 }
 
 func getPodInfoBySelf(path string) (podId string, containerId string) {
	 f, err := os.Open(path)
	 if err != nil {
		 return "", ""
	 }
	 d, err := ioutil.ReadAll(f)
	 if err != nil {
		 return "", ""
	 }
	 lines := strings.Split(string(d), "\n")
	 var rawLine string
	 // find line name=systemd
	 // /proc/self/cgroup
	 // 1:name=systemd:/system.slice/containerd.service/kubepods-pod3afbda42_dabf_482d_962e_77bada079c54.slice:cri-containerd:68ac51f452910c79d29c2f16d5130432d30e94b890069195d8b2381b88e11489
	 for _, line := range lines {
		 if strings.Contains(line, "name=systemd") {
			 rawLine = line
			 break
		 }
	 }
	 // find pod ID and container ID
	 raws := strings.SplitN(rawLine, "-pod", 2)
	 if len(raws) == 2 {
		 ids := strings.SplitN(raws[1], ".slice:cri-containerd:", 2)
		 if len(ids) == 2 {
			 return normalize(ids[0]), ids[1]
		 }
	 }
	 return "", ""
 }
 
 func main() {
	 cmdFlags := pflag.CommandLine
 
	 cmdFlags.StringVar(&addr, "addr", "", "RPC address location for dial")
	 cmdFlags.StringVar(&busID, "bus-id", "", "GPU card bus id of caller")
	 cmdFlags.StringVar(&podUID, "pod-uid", "", "Pod UID of caller")
	 cmdFlags.StringVar(&contName, "cont-name", "", "Container name of caller")
	 cmdFlags.StringVar(&contID, "cont-id", "", "Container id of calller")
 
	 flags.InitFlags()
	 goflag.CommandLine.Parse([]string{})
	 logs.InitLogs()
	 defer logs.FlushLogs()
 
	 if len(addr) == 0 || len(podUID) == 0 || (len(contName) == 0 && len(contID) == 0) {
		 klog.Fatalf("argument is empty, current: %s", cmdFlags.Args())
	 }
 
	 conn, err := grpc.Dial(addr, utils.DefaultDialOptions...)
	 if err != nil {
		 klog.Fatalf("can't dial %s, error %v", addr, err)
	 }
	 defer conn.Close()
 
	 client := vcudaapi.NewVCUDAServiceClient(conn)
	 ctx := context.TODO()
	 podUID, contID = getPodInfoBySelf("/proc/self/cgroup")
 
	 req := &vcudaapi.VDeviceRequest{
		 BusId:         busID,
		 PodUid:        podUID,
		 ContainerName: contName,
	 }
 
	 if len(contID) > 0 {
		 req.ContainerName = ""
		 req.ContainerId = contID
	 }
 
	 _, err = client.RegisterVDevice(ctx, req)
	 if err != nil {
		 klog.Fatalf("fail to get response from manager, error %v", err)
	 }
 }
 