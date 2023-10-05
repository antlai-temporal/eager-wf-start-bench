package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/pborman/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/antlai-temporal/eager-wf-start-bench/eagerbench"
)

const TaskQueueName = "eager-wf-bench"
const ClientKeyPath = "./client.key"
const ClientCertPath = "./client.pem"
const HostPort = "eager-wf-start.a2dd6.tmprl.cloud:7233"
const Namespace = "eager-wf-start.a2dd6"

func computeStats(input []float64) (float64, float64, float64, float64) {
	p50, err := stats.Median(input)
	if err != nil {
		log.Fatalln("Cannot compute Percentile", err)
	}
	p90, err := stats.Percentile(input, 90.0)
	if err != nil {
		log.Fatalln("Cannot compute Percentile", err)
	}
	p95, err := stats.Percentile(input, 95.0)
	if err != nil {
		log.Fatalln("Cannot compute Percentile", err)
	}
	p99, err := stats.Percentile(input, 99.0)
	if err != nil {
		log.Fatalln("Cannot compute Percentile", err)
	}
	return p50, p90, p95, p99
}

func executeOnce(c client.Client, enableEagerStart bool) (time.Duration, time.Duration) {
	workflowOptions := client.StartWorkflowOptions{
		ID:               "eager_wf_" + uuid.New(),
		TaskQueue:        TaskQueueName,
		EnableEagerStart: enableEagerStart,
	}
	ctx := context.Background()
	now := time.Now()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, eagerbench.Workflow, now)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}
	var tInteract time.Duration
	err = we.Get(ctx, &tInteract)
	if err != nil {
		log.Fatalln("Cannot get time to interact", err)
	}

	tComplete := time.Since(now)
	return tInteract, tComplete
}

func main() {
	enableEagerStartPtr := flag.Bool("eager", false, "Enable Eager Start")
	isLocalPtr := flag.Bool("local", false, "Use a local server")
	numIterPtr := flag.Int("iter", 100, "Number of workflows started")
	flag.Parse()
	log.Println("enableEagerStart is", *enableEagerStartPtr, " #workflows is", *numIterPtr, " local is", *isLocalPtr)

	var tInteractAll, tCompleteAll []float64
	var c client.Client
	var err error
	if *isLocalPtr {
		c, err = client.Dial(client.Options{})
	} else {
		cert, errKey := tls.LoadX509KeyPair(ClientCertPath, ClientKeyPath)
		if errKey != nil {
			log.Fatalln("Unable to load cert and key pair.", errKey)
		}
		c, err = client.Dial(client.Options{
			HostPort:  HostPort,
			Namespace: Namespace,
			ConnectionOptions: client.ConnectionOptions{
				TLS: &tls.Config{Certificates: []tls.Certificate{cert}},
			},
		})
	}
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, TaskQueueName, worker.Options{})

	w.RegisterWorkflow(eagerbench.Workflow)
	w.RegisterActivity(eagerbench.Activity)
	err = w.Start()
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
	defer w.Stop()
	log.Println("Time_to_interact (microseconds), Time_to_complete (microseconds)")

	// Warm up
	for i := 0; i < 50; i++ {
		executeOnce(c, *enableEagerStartPtr)
		//		time.Sleep(100 * time.Millisecond)
	}

	for i := 0; i < *numIterPtr; i++ {
		tInteract, tComplete := executeOnce(c, *enableEagerStartPtr)
		tInteractAll = append(tInteractAll, float64(tInteract.Microseconds()))
		tCompleteAll = append(tCompleteAll, float64(tComplete.Microseconds()))
		log.Println(tInteract.Microseconds(), tComplete.Microseconds())
		//		time.Sleep(100 * time.Millisecond)
	}

	p50, p90, p95, p99 := computeStats(tInteractAll)
	log.Println("Time_to_interact (microseconds): p50=", p50, " p90=", p90, " p95=", p95, " p99=", p99)

	p50, p90, p95, p99 = computeStats(tCompleteAll)
	log.Println("Time_to_complete (microseconds): p50=", p50, " p90=", p90, " p95=", p95, " p99=", p99)

}
