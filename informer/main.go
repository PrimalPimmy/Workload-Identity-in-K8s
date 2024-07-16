package main

import (
    "log"
    "os"
    "syscall"
    "time"
    "fmt"
    "os/exec"
    "strconv"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/dynamic/dynamicinformer"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
)

func main() {
    // Create in-cluster config
    config, err := rest.InClusterConfig()
    if err != nil {
        log.Fatalf("Error creating in-cluster config: %v", err)
    }

    // Create dynamic client
    dynamicClient, err := dynamic.NewForConfig(config)
    if err != nil {
        log.Fatalf("Error creating dynamic client: %v", err)
    }

    // Create dynamic informer factory
    factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, time.Minute, "spire", nil)

    // Create informer for ConfigMaps
    gvr := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
    informer := factory.ForResource(gvr).Informer()

    // Add event handler
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        UpdateFunc: func(oldObj, newObj interface{}) {
            newConfigMap := newObj.(metav1.Object)
            if newConfigMap.GetName() == "clusters" {
                log.Println("clusters ConfigMap updated. Sending SIGUSR1 to spire-server...")
                sendSignalToSpireServer()
            }
        },
    })

    // Start informer
    stopCh := make(chan struct{})
    factory.Start(stopCh)
    factory.WaitForCacheSync(stopCh)

    // Wait forever
    select {}
}

func sendSignalToSpireServer() {
    // Find the PID of the spire-server process
    pid, err := findSpireServerPID()
    if err != nil {
        log.Printf("Error finding spire-server PID: %v", err)
        return
    }

    // Send SIGUSR1 to the spire-server process
    process, err := os.FindProcess(pid)
    if err != nil {
        log.Printf("Error finding process: %v", err)
        return
    }

    err = process.Signal(syscall.SIGUSR1)
    if err != nil {
        log.Printf("Error sending SIGUSR1 signal: %v", err)
        return
    }

    log.Println("SIGUSR1 signal sent to spire-server")
}

func findSpireServerPID() (int, error) {
    cmd := exec.Command("pgrep", "spire-server")
    output, err := cmd.Output()
    if err != nil {
        fmt.Println("Error finding spire-server process:", err)
    }

    pid, err := strconv.Atoi(string(output[:len(output)-1])) // Remove newline
    if err != nil {
        fmt.Println("Error converting PID to integer:", err)
    }
	return pid, nil
}

