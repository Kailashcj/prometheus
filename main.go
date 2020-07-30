package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
)

const addr = ":2000"

type metricsCollection struct {
	jobSuccess prometheus.Counter
	jobFailure prometheus.Counter
}

var metrics *metricsCollection //variable defination

func fiboRec(n int) int {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	return fiboRec(n-1) + fiboRec(n-2)
}

func runJob() {

	//initialization of variable metrics
	metrics = &metricsCollection{
		jobSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "fibo_total_success_counts",
			Help: "The total number of successfull calculations",
		}),
		jobFailure: promauto.NewCounter(prometheus.CounterOpts{
			Name: "fibo_total_failure_counts",
			Help: "The total number of failed calculations",
		}),
	}
	var number, output, failedCount int
	rand.Seed(time.Now().UnixNano())
	go func() {
		for {
			jobResult := "success"
			number = rand.Intn(58)
			output = fiboRec(number - 5)
			switch {
			case output <= 0 && failedCount < 5:
				{
					jobResult = "failed"
					failedCount++
					metrics.jobFailure.Inc()
				}
			default:
				{
					metrics.jobSuccess.Inc()
				}
			}
			fmt.Println(number, output, jobResult)
			if failedCount == 1 {
				break
			}
			time.Sleep(3 * time.Second)
		}
	}()
}

func main() {
	runJob()
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())

	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	errCh := make(chan error)
	// Start server
	go func() {
		errCh <- http.ListenAndServe(addr, r)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Wait for signal or error.
	log.Info("listening", zap.String("address", addr))
	select {
	case sig := <-signals:
		log.Info("caught signal, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		log.Error("serve error, shutting down", zap.Error(err))
	}

	// shutdown http server
	const timeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("shutdown metrics http server", zap.Error(err))
	}

	//http.Handle("/metrics", promhttp.Handler())
	fmt.Println("main routine listening")
	//http.ListenAndServe(":2000", nil)

}
