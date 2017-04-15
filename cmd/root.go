package cmd

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/net"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "fpsutil",
	Short: "Very basic network IO counters",
	Long:  `Very basic network IO counters`,
	Run:   mainLoop,
}

func mainLoop(cmd *cobra.Command, args []string) {

	timestamp := time.Now()
	filename := fmt.Sprintf("netstats%s.csv", timestamp.Format("20060102T150405"))
	f, err := os.Create(filename)
	check(err)
	defer f.Close()

	f.WriteString("Timestamp,Bytes Total/sec,Bytes Sent/sec,Bytes Received/sec,Packets Total/sec,Packets Sent/sec,Packets Received/sec,Total number of errors while receiving/sec,Total number of errors while sending/sec,Total number of dropped incoming packets/sec,Total number of dropped outgoing packets/sec,Total number of FIFO buffers errors while receiving/sec,Total number of FIFO buffers errors while sending/sec\n")
	f.Sync()

	countersChannel := make(chan string, 60)
	defer close(countersChannel)

	buffer := bufio.NewWriter(f)
	go logger(countersChannel, buffer)

	ticker := time.NewTicker(time.Second)
	prevStats, _ := getTotalIOCountersStat()
	for t := range ticker.C {
		nextStats, _ := getTotalIOCountersStat()
		str := fmt.Sprintf("%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
			t.Format("20060102T15:04:05.999-0700"),
			nextStats.BytesSent-prevStats.BytesSent+nextStats.BytesRecv-prevStats.BytesRecv,
			nextStats.BytesSent-prevStats.BytesSent,
			nextStats.BytesRecv-prevStats.BytesRecv,
			nextStats.PacketsSent-prevStats.PacketsSent+nextStats.PacketsRecv-prevStats.PacketsRecv,
			nextStats.PacketsSent-prevStats.PacketsSent,
			nextStats.PacketsRecv-prevStats.PacketsRecv,
			nextStats.Errin-prevStats.Errin,
			nextStats.Errout-prevStats.Errout,
			nextStats.Dropin-prevStats.Dropin,
			nextStats.Dropout-prevStats.Dropout,
			nextStats.Fifoin-prevStats.Fifoin,
			nextStats.Fifoout-prevStats.Fifoout)
		prevStats = nextStats
		countersChannel <- str
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func logger(msgs <-chan string, buffer *bufio.Writer) {
	const flushInterval int = 1024
	var buffered int
	for msg := range msgs {
		n, _ := buffer.WriteString(msg)
		buffered += n
		if buffered > flushInterval {
			buffer.Flush()
			buffered = 0
		}
	}
	buffer.Flush()
}

func getTotalIOCountersStat() (net.IOCountersStat, error) {
	ioCountersStats, err := net.IOCounters(false)
	return ioCountersStats[0], err
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fpsutil.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".fpsutil") // name of config file (without extension)
	viper.AddConfigPath("$HOME")    // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
