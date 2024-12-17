package main

import (
	"database/sql"
	"flag"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	// 定义命令行参数
	dsn := flag.String("dsn", "", "PostgreSQL DSN")
	pushURL := flag.String("push-url", "", "Pushgateway URL")
	interval := flag.Duration("interval", time.Minute, "Interval between queries")
	jobName := flag.String("job", "oula_block_height", "Job name for Pushgateway")

	flag.Parse()

	if *dsn == "" {
		log.Fatal("DSN must be provided")
	}

	log.Printf("Starting program with DSN: %s, Pushgateway URL: %s, Interval: %s, Job Name: %s", *dsn, *pushURL, *interval, *jobName)

	// 连接到数据库
	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()
	log.Println("Successfully connected to the database")

	// 定义 Prometheus Gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "oula_aleo_block_max_height",
		Help: "The maximum block height",
	})

	// 定期查询数据库并推送数据到 Pushgateway
	for {
		log.Println("Querying the database for the maximum block height...")
		var maxHeight int
		err := db.QueryRow("SELECT MAX(height) FROM block").Scan(&maxHeight)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
		} else {
			log.Printf("Query successful, maximum block height: %d", maxHeight)
			gauge.Set(float64(maxHeight))
			pushMetrics(*pushURL, *jobName, gauge)
		}

		log.Printf("Sleeping for %s before the next query", *interval)
		time.Sleep(*interval)
	}
}

// pushMetrics 使用 Pushgateway 推送数据
func pushMetrics(pushURL, jobName string, gauge prometheus.Gauge) {
	log.Printf("Pushing metrics to Pushgateway at %s with job name %s...", pushURL, jobName)
	if err := push.New(pushURL, jobName).
		Collector(gauge).
		Push(); err != nil {
		log.Printf("Could not push to Pushgateway: %v", err)
	} else {
		log.Println("Pushed metrics successfully")
	}
}
